package fetch

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/dataexport"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/utils"
	"github.com/cockroachdb/molt/verify/dbverify"
	"github.com/cockroachdb/molt/verify/rowverify"
	"github.com/cockroachdb/molt/verify/tableverify"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

const (
	bucketForTest = `migrations-fetch-ci-test`
	pathInBucket  = `failed-export`

	localPathForTest = `/tmp/failedexport`
)

func genDataStore(
	ctx context.Context, logger zerolog.Logger, storageType string, conns dbconn.OrderedConns,
) (datablobstorage.Store, error) {
	var datastorePayload any
	switch storageType {
	case "gcp":
		datastorePayload = &datablobstorage.GCPPayload{
			GCPBucket:  bucketForTest,
			BucketPath: pathInBucket,
		}
	case "aws":
		datastorePayload = &datablobstorage.S3Payload{
			S3Bucket:   bucketForTest,
			BucketPath: pathInBucket,
			Region:     "us-east-1",
		}
	case "local":
		localStoreListenAddr, localStoreCrdbAccessAddr := testutils.GetLocalStoreAddrs("crdb", "4541")
		datastorePayload = &datablobstorage.LocalPathPayload{
			LocalPath:               localPathForTest,
			LocalPathListenAddr:     localStoreListenAddr,
			LocalPathCRDBAccessAddr: localStoreCrdbAccessAddr,
		}
	case "directcopy":
		datastorePayload = &datablobstorage.DirectCopyPayload{
			TargetConnForCopy: conns[1].(*dbconn.PGConn).Conn,
		}
	default:
		return nil, errors.Newf("unknown storage type: %s", storageType)
	}

	return datablobstorage.GenerateDatastore(ctx, datastorePayload, logger, true /* TestFailedWriteToBucket */, true /* TestOnly */)
}

func TestFailedWriteToStore(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stderr)
	const (
		dbName     = "failed_fetch_export_table"
		tableName  = "testfailedexport"
		schemaName = "public"

		createTableQuery        = `CREATE TABLE testfailedexport (x int PRIMARY KEY)`
		insertTableQueryPattern = `INSERT INTO testfailedexport VALUES (%d)`
	)

	for _, storageType := range []struct {
		name        string
		expectError string
	}{
		{
			name:        "gcp",
			expectError: datablobstorage.GCPWriterMockErrMsg,
		},
		{
			name:        "aws",
			expectError: datablobstorage.AWSUploadFileMockErrMsg,
		},
		{
			name:        "local",
			expectError: datablobstorage.LocalWriterMockErrMsg,
		},
		{
			name:        "directcopy",
			expectError: datablobstorage.DirectCopyWriterMockErrMsg,
		},
	} {
		t.Run(storageType.name, func(t *testing.T) {
			type namedTb struct {
				testutils.FetchTestingKnobs
				decription string
			}

			tbs := []namedTb{
				{
					decription: "after read from pipe",
					FetchTestingKnobs: testutils.FetchTestingKnobs{
						FailedWriteToBucket: testutils.FailedWriteToBucketKnob{
							FailedAfterReadFromPipe: true,
						},
					},
				},
			}

			if storageType.name == "local" || storageType.name == "directcopy" {
				tbs = append(tbs, namedTb{
					decription: "before read from pipe",
					FetchTestingKnobs: testutils.FetchTestingKnobs{
						FailedWriteToBucket: testutils.FailedWriteToBucketKnob{
							FailedBeforeReadFromPipe: true,
						},
					},
				})
			}

			for _, tb := range tbs {
				t.Run(tb.decription, func(t *testing.T) {
					for _, tc := range []struct {
						description  string
						tableRows    int
						rowBatchSize int
						flushRows    int
						noError      bool
					}{
						{
							description:  "empty table",
							tableRows:    0,
							rowBatchSize: 10,
							flushRows:    10,
							noError:      true,
						},
						{
							description:  "one row table",
							tableRows:    1,
							rowBatchSize: 10,
							flushRows:    10,
						},
						{
							description:  "multi batch export",
							tableRows:    20,
							rowBatchSize: 3,
							flushRows:    3,
						},
					} {
						t.Run(tc.description, func(t *testing.T) {

							var conns dbconn.OrderedConns
							var err error
							conns[0], err = dbconn.TestOnlyCleanDatabase(ctx, "source", testutils.MySQLConnStr(), dbName)
							require.NoError(t, err)
							conns[1], err = dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
							require.NoError(t, err)

							// Check the 2 dbs are up.
							for _, c := range conns {
								_, err := testutils.ExecConnQuery(ctx, "SELECT 1", c)
								require.NoError(t, err)
							}

							defer func() {
								require.NoError(t, conns[0].Close(ctx))
								require.NoError(t, conns[1].Close(ctx))
							}()

							// Create the table in the both databases.
							for _, conn := range conns {
								_, err := testutils.ExecConnQuery(ctx, createTableQuery, conn)
								require.NoError(t, err)
							}

							// Insert rows only on the source database table.
							for d := 1; d <= tc.tableRows; d++ {
								_, err := testutils.ExecConnQuery(ctx, fmt.Sprintf(insertTableQueryPattern, d), conns[0])
								require.NoError(t, err)
							}

							// Preparation for the fetch.exportTable function.
							fetchCfg := Config{
								FlushRows: tc.flushRows,
								Cleanup:   false,
								ExportSettings: dataexport.Settings{
									RowBatchSize: tc.rowBatchSize,
								},
							}

							dbTables, err := dbverify.Verify(ctx, conns)
							require.NoError(t, err)
							dbTables, err = utils.FilterResult(utils.FilterConfig{
								SchemaFilter: schemaName,
								TableFilter:  tableName,
							}, dbTables)

							require.NoError(t, err)

							tables, err := tableverify.VerifyCommonTables(ctx, conns, logger, dbTables.Verified)
							require.NoError(t, err)
							require.Equal(t, 1, len(tables))
							verifiedTable := tables[0].VerifiedTable

							sqlSrc, err := dataexport.InferExportSource(ctx, fetchCfg.ExportSettings, conns[0])
							require.NoError(t, err)

							dataSrc, err := genDataStore(ctx, logger, storageType.name, conns)
							require.NoError(t, err)

							_, err = exportTable(
								ctx,
								fetchCfg,
								logger,
								sqlSrc,
								dataSrc,
								verifiedTable,
								rowverify.TableShard{
									ShardNum:    1,
									StartPKVals: []tree.Datum{tree.NewDInt(tree.DInt(1))},
									EndPKVals:   []tree.Datum{},
								},
								tb.FetchTestingKnobs,
							)

							if tc.noError {
								require.NoError(t, err)
							} else {
								require.ErrorContains(t, err, storageType.expectError)
							}

							t.Logf("test passed!")
						})

					}
				})
			}
		})
	}
}
