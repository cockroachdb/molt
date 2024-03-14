package fetch

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/cockroachdb/datadriven"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/dataexport"
	"github.com/cockroachdb/molt/fetch/status"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/utils"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

type storeDetails struct {
	scheme  string
	host    string
	subpath string
	url     *url.URL
}
type compType int

const (
	none compType = iota
	longer
	shorter
)

type exportFailureCase int

const (
	noneFailedWhenExport exportFailureCase = iota
	failedWhenExportDataToPipe
	failedWhenInitNewWriter
	failedWhenWriteToCSV
)

type elapsedComparison struct {
	elapsed time.Duration
	comp    compType
}

var dockerInternalRegex = regexp.MustCompile(`host\.docker\.internal`)

// This is needed because tests are usually recorded on MacOS, which will use host.docker.internal.
// However in CI it tries to use localhost. We enforce this so that we normalize it
// to localhost for recorded data.
func replaceDockerInternalLocalHost(input string) string {
	return dockerInternalRegex.ReplaceAllString(input, "localhost")
}

func TestDataDriven(t *testing.T) {
	for _, tc := range []struct {
		desc string
		path string
		src  string
		dest string
	}{
		{desc: "pg", path: "testdata/pg", src: testutils.PGConnStr(), dest: testutils.CRDBConnStr()},
		{desc: "mysql", path: "testdata/mysql", src: testutils.MySQLConnStr(), dest: testutils.CRDBConnStr()},
		{desc: "crdb", path: "testdata/crdb", src: testutils.CRDBConnStr(), dest: testutils.CRDBTargetConnStr()},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			datadriven.Walk(t, tc.path, func(t *testing.T, path string) {
				ctx := context.Background()
				var conns dbconn.OrderedConns
				var err error
				dbName := "fetch_" + tc.desc + "_" + strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
				logger := zerolog.New(os.Stderr)

				conns[0], err = dbconn.TestOnlyCleanDatabase(ctx, "source", tc.src, dbName)
				require.NoError(t, err)
				conns[1], err = dbconn.TestOnlyCleanDatabase(ctx, "target", tc.dest, dbName)
				require.NoError(t, err)

				for _, c := range conns {
					_, err := testutils.ExecConnQuery(ctx, "SELECT 1", c)
					require.NoError(t, err)
				}
				t.Logf("successfully connected to both source and target")

				datadriven.RunTest(t, path, func(t *testing.T, d *datadriven.TestData) (ddtRes string) {
					// Extract common arguments.
					args := d.CmdArgs[:0]
					var expectError bool
					var suppressErrorMessage bool
					for _, arg := range d.CmdArgs {
						switch arg.Key {
						case "expect-error":
							expectError = true
						case "suppress-error":
							suppressErrorMessage = true
						default:
							args = append(args, arg)
						}
					}
					d.CmdArgs = args

					switch d.Cmd {
					case "create-schema-stmt":
						if len(d.CmdArgs) == 0 {
							t.Errorf("table filter not specified")
						}
						showDroppedConstraints := false
						for _, arg := range d.CmdArgs {
							if arg.Key == "show-dropped-constraints" && arg.Vals[0] == "true" {
								showDroppedConstraints = true
							}
						}
						return func() string {
							var stmts []string
							tableName := d.CmdArgs[0]
							_, createTableErr := testutils.ExecConnQuery(ctx, d.Input, conns[0])
							if createTableErr != nil {
								if expectError {
									stmts = append(stmts, createTableErr.Error())
									return strings.Join(stmts, "\n")
								}
								require.NoError(t, createTableErr)
							}

							defer func() {
								_, dropTableErr := testutils.ExecConnQuery(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tableName.String()), conns[0])
								require.NoError(t, dropTableErr)
							}()
							tableFilter := utils.FilterConfig{TableFilter: tableName.String()}
							missingTables, err := getFilteredMissingTables(ctx, conns, tableFilter)
							require.NoError(t, err)

							for _, missingTable := range missingTables {
								srcConn := conns[0]
								stmt, err := GetCreateTableStmt(ctx, logger, srcConn, missingTable.DBTable)
								if err != nil {
									stmts = append(stmts, err.Error())
									// Somehow we need to recreate the connection, otherwise pg will show "conn busy" error.
									newConn, err := srcConn.Clone(ctx)
									require.NoError(t, err)
									require.NoError(t, srcConn.Close(ctx))
									conns[0] = newConn
								} else {
									stmts = append(stmts, stmt)
								}
								if showDroppedConstraints {
									stmts = append(stmts, `------ DROPPED CONSTRAINTS ------`)
									droppedConstraints, err := GetConstraints(ctx, logger, conns[0], missingTable.DBTable)
									if err != nil {
										stmts = append(stmts, err.Error())
									} else {
										stmts = append(stmts, droppedConstraints...)
									}
								}
							}
							return strings.Join(stmts, "\n")
						}()
					case "exec":
						return testutils.ExecConnTestdata(t, d, conns)
					case "query":
						return replaceDockerInternalLocalHost(testutils.QueryConnCommand(t, d, conns))
					case "fetch":
						filter := utils.DefaultFilterConfig()
						truncate := true
						live := false
						direct := false
						compress := false
						corruptCSVFile := false
						failedEstablishConnForExport := false
						fetchId := ""
						passedInDir := ""
						cleanup := false
						continuationToken := ""
						overrideFile := ""
						flushRows := 0
						dropAndRecreateSchema := false
						createFiles := []string{}
						bucketPath := ""
						sDetails := storeDetails{}
						numShards := 1
						elapsedComp := elapsedComparison{}
						var fetchStartTime time.Time
						var fetchFinishTime time.Time
						var failedShardIdx *int
						var failedIterNum *int
						var failedRowCnt *int
						var failedWhenExportType exportFailureCase

						var showFetchElapsed bool

						knobs := testutils.FetchTestingKnobs{}

						for _, cmd := range d.CmdArgs {
							switch cmd.Key {
							case "live":
								live = true
							case "notruncate":
								truncate = false
							case "direct":
								direct = true
							case "compress":
								compress = true
							case "corrupt-csv":
								corruptCSVFile = true
							case "failed-conn-export":
								failedEstablishConnForExport = true
							case "fetch-id":
								fetchId = cmd.Vals[0]
							case "store-dir":
								passedInDir = cmd.Vals[0]
							case "cleanup-dir":
								cleanup = true
							case "continuation-token":
								continuationToken = cmd.Vals[0]
							case "drop-and-recreate-schema":
								dropAndRecreateSchema = true
								truncate = false
							case "override-file":
								overrideFile = cmd.Vals[0]
							case "flush-rows":
								flushRowsAtoi, err := strconv.Atoi(cmd.Vals[0])
								require.NoError(t, err)
								flushRows = flushRowsAtoi
							case "create-files":
								createFiles = strings.Split(cmd.Vals[0], ",")
							case "show-fetch-elapsed":
								showFetchElapsed = true
							case "export-mode":
								expMdStr := cmd.Vals[0]
								switch strings.ToLower(expMdStr) {
								case "select":
									knobs.ExpMode = testutils.ExportWithSelect
								case "copy":
									knobs.ExpMode = testutils.ExportWithCopy
								default:
									t.Fatalf("unknown export mode: %s", expMdStr)
								}
							case "bucket-path":
								bucketPath = cmd.Vals[0]
								url, err := url.Parse(bucketPath)
								require.NoError(t, err)
								subPath := strings.TrimPrefix(url.Path, "/")
								host := url.Host

								sDetails = storeDetails{
									scheme:  url.Scheme,
									host:    host,
									subpath: subPath,
									url:     url,
								}
							case "shards":
								s := cmd.Vals[0]
								numShards, err = strconv.Atoi(s)
								require.NoError(t, err)
							case "fetch-elapsed":
								elapsedRequirement := cmd.Vals[0]
								if strings.HasPrefix(elapsedRequirement, ">") {
									elapsedComp.comp = longer
								} else if strings.HasPrefix(elapsedRequirement, "<") {
									elapsedComp.comp = shorter
								} else {
									t.Fatalf("the elapsed comparison must be > or <, but got: %q", cmd.Vals[0])
								}
								elapsedComp.elapsed, err = time.ParseDuration(elapsedRequirement[1:])
								require.NoError(t, err)
							case "failed-export-type":
								switch cmd.Vals[0] {
								case "export-to-pipe":
									failedWhenExportType = failedWhenExportDataToPipe
								case "init-new-writer":
									failedWhenExportType = failedWhenInitNewWriter
								case "write-to-csv":
									failedWhenExportType = failedWhenWriteToCSV
								default:
									t.Fatalf("export failure type must be one of {export-to-pipe|init-new-writer|write-to-csv}")
								}
							case "failed-shard-idx":
								require.Greater(t, len(cmd.Vals), 0)
								shardIdx, err := strconv.Atoi(cmd.Vals[0])
								require.NoError(t, err)
								failedShardIdx = &shardIdx
							case "failed-iter-idx":
								require.Greater(t, len(cmd.Vals), 0)
								iterIdx, err := strconv.Atoi(cmd.Vals[0])
								require.NoError(t, err)
								failedIterNum = &iterIdx
							case "failed-row-cnt":
								require.Greater(t, len(cmd.Vals), 0)
								rowCnt, err := strconv.Atoi(cmd.Vals[0])
								require.NoError(t, err)
								failedRowCnt = &rowCnt

							default:
								t.Errorf("unknown key %s", cmd.Key)
							}
						}

						dir := ""
						if passedInDir == "" {
							createDir, err := os.MkdirTemp("", "")
							require.NoError(t, err)
							dir = createDir
						} else {
							dir = passedInDir
						}

						// Create mock files with invalid data.
						if len(createFiles) > 0 {
							for _, file := range createFiles {
								require.NoError(t, createAndWriteDummyData(dir, file))
							}
						}

						var src datablobstorage.Store
						defer func() {
							if src != nil {
								require.NoError(t, src.Cleanup(ctx))
							}
						}()
						if direct {
							src = datablobstorage.NewCopyCRDBDirect(logger, conns[1].(*dbconn.PGConn).Conn)
						} else if bucketPath != "" {
							switch sDetails.scheme {
							case "s3", "S3":
								sess := createS3Bucket(t, ctx, sDetails)
								src = datablobstorage.NewS3Store(logger, sess, credentials.Value{}, sDetails.host, sDetails.subpath, true)
							case "gs", "GS":
								gcpClient := createGCPBucket(t, ctx, sDetails)
								src = datablobstorage.NewGCPStore(logger, gcpClient, nil, sDetails.host, sDetails.subpath, true)
							default:
								require.Contains(t, []string{"s3", "S3", "gs", "GS"}, sDetails.scheme)
							}
						} else {
							t.Logf("stored in local dir %q", dir)

							localStoreListenAddr, localStoreCrdbAccessAddr := testutils.GetLocalStoreAddrs(tc.desc, "4040")

							src, err = datablobstorage.NewLocalStore(logger, dir, localStoreListenAddr, localStoreCrdbAccessAddr)
							require.NoError(t, err)
						}

						compressionFlag := compression.None
						if compress {
							compressionFlag = compression.GZIP
						}

						if corruptCSVFile {
							knobs.TriggerCorruptCSVFile = true
						}
						if failedEstablishConnForExport {
							knobs.FailedEstablishSrcConnForExport = true
						}

						switch failedWhenExportType {
						case failedWhenExportDataToPipe:
							if failedShardIdx == nil {
								t.Fatalf("failed sharded idx is not specified")
							}
							knobs.FailedToExportForShard = &testutils.FailedToExportForShardKnob{
								FailedExportDataToPipeCondition: func(table tree.Name, shardIdx int) bool {
									if filter.TableFilter != utils.DefaultFilterString && table.String() != filter.TableFilter {
										return false
									}
									return shardIdx == *failedShardIdx
								},
							}
						case failedWhenInitNewWriter:
							if failedShardIdx == nil {
								t.Fatalf("failed sharded idx is not specified")
							}
							if failedIterNum == nil {
								t.Fatalf("failed iteration idx is not specified")
							}
							knobs.FailedToExportForShard = &testutils.FailedToExportForShardKnob{
								FailedReadDataFromPipeInitWriterCondition: func(table tree.Name, shardIdx int, itNum int) bool {
									if filter.TableFilter != utils.DefaultFilterString && table.String() != filter.TableFilter {
										return false
									}
									return shardIdx == *failedShardIdx && itNum == *failedIterNum
								},
							}
						case failedWhenWriteToCSV:
							knobs.FailedToExportForShard = &testutils.FailedToExportForShardKnob{
								FailedReadDataFromPipeWriteToCSVWriterCondition: func(table tree.Name, shardIdx int, rowCnt int) bool {
									if filter.TableFilter != utils.DefaultFilterString && table.String() != filter.TableFilter {
										return false
									}
									return shardIdx == *failedShardIdx && rowCnt == *failedRowCnt
								},
							}
						default:
						}

						if elapsedComp.elapsed != 0 || showFetchElapsed {
							fetchStartTime = time.Now()
						}

						err = Fetch(
							ctx,
							Config{
								Live:                     live,
								Truncate:                 truncate,
								DropAndRecreateNewSchema: dropAndRecreateSchema,
								ExportSettings: dataexport.Settings{
									RowBatchSize: 2,
								},
								Compression:          compressionFlag,
								FetchID:              fetchId,
								Cleanup:              cleanup,
								ContinuationToken:    continuationToken,
								ContinuationFileName: overrideFile,
								FlushRows:            flushRows,
								NonInteractive:       true,
								Shards:               numShards,
							},
							logger,
							conns,
							src,
							filter,
							knobs,
						)

						// We want a more thorough cleanup if we want to cleanup dir.
						// This makes it so that we ensure we have a fresh environment.
						defer func() {
							if cleanup {
								err := os.RemoveAll(dir)
								require.NoError(t, err)
							}
						}()

						if elapsedComp.elapsed != 0 || showFetchElapsed {
							fetchFinishTime = time.Now()
							actualElapsed := fetchFinishTime.Sub(fetchStartTime)
							if showFetchElapsed {
								defer func() {
									ddtRes = strings.Join([]string{fmt.Sprintf("elapsed:%s", actualElapsed), ddtRes}, "\n")
								}()
							}

							if elapsedComp.elapsed != 0 {
								if elapsedComp.comp == longer {
									require.GreaterOrEqual(t, actualElapsed, elapsedComp.elapsed)
								} else if elapsedComp.comp == shorter {
									require.LessOrEqual(t, actualElapsed, elapsedComp.elapsed)
								} else {
									t.Fatalf("elapsed duration comparison must be > or <")
								}
							}
						}

						if expectError && !suppressErrorMessage {
							require.Error(t, err)
							return replaceDockerInternalLocalHost(err.Error())
						} else if expectError && suppressErrorMessage {
							require.Error(t, err)
							return ""
						}
						require.NoError(t, err)
						return ""
					case "list-tokens":
						// We don't want to clean the database in this case.
						targetConn := conns[1]
						targetPgConn, valid := targetConn.(*dbconn.PGConn)
						require.Equal(t, true, valid)

						numResults := 5

						for _, cmd := range d.CmdArgs {
							switch cmd.Key {
							case "num-results":
								res, err := strconv.Atoi(cmd.Vals[0])
								require.NoError(t, err)
								numResults = res
							default:
								t.Errorf("unknown key %s", cmd.Key)
							}
						}

						val, err := ListContinuationTokens(ctx, true /*testOnly*/, targetPgConn.Conn, numResults)

						if !expectError {
							require.NoError(t, err)
							return val
						} else {
							require.Error(t, err)
							return err.Error()
						}

					default:
						t.Errorf("unknown command: %s", d.Cmd)
					}

					return ""
				})
			})
		})
	}
}

func createAndWriteDummyData(dir, fileName string) (retErr error) {
	f, err := os.Create(path.Join(dir, fileName))
	defer func() {
		err := f.Close()
		if err != nil {
			retErr = errors.Wrap(err, retErr.Error())
		}
	}()

	if err != nil {
		return err
	}
	_, err = f.WriteString("invalid\ndata")
	if err != nil {
		return err
	}

	return nil
}

func createS3Bucket(t *testing.T, ctx context.Context, sDetails storeDetails) *session.Session {
	config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials("test", "test", ""),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         aws.String("http://s3.localhost.localstack.cloud:4566"),
		Region:           aws.String("us-east-1"),
	}
	sess, err := session.NewSession(config)
	require.NoError(t, err)
	s3Cli := s3.New(sess)
	_, err = s3Cli.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(sDetails.host),
	})
	require.NoError(t, err)
	return sess
}

func createGCPBucket(t *testing.T, ctx context.Context, sDetails storeDetails) *storage.Client {
	gcpClient, err := storage.NewClient(ctx,
		option.WithEndpoint("http://localhost:4443/storage/v1/"),
		option.WithoutAuthentication(),
	)

	require.NoError(t, err)

	// Create the test bucket
	bucket := gcpClient.Bucket(sDetails.host)
	if _, err := bucket.Attrs(ctx); err == nil {
		// Skip creating the bucket.
		fmt.Printf("skipping creation of bucket %s because it already exists\n", sDetails.host)
		return gcpClient
	}
	err = bucket.Create(ctx, "", nil)
	require.NoError(t, err)
	return gcpClient
}

func TestInitStatusEntry(t *testing.T) {
	ctx := context.Background()
	dbName := "fetch_test_status"

	t.Run("successfully initialized when tables not created", func(t *testing.T) {
		conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
		require.NoError(t, err)
		pgConn := conn.(*dbconn.PGConn).Conn

		actual, err := initStatusEntry(ctx, Config{}, pgConn, "PostgreSQL")
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, actual.ID)
	})

	t.Run("successfully initialized when tables created beforehand", func(t *testing.T) {
		conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
		require.NoError(t, err)
		pgConn := conn.(*dbconn.PGConn).Conn
		// Setup the tables that we need to write for status.
		require.NoError(t, status.CreateStatusAndExceptionTables(ctx, pgConn))

		actual, err := initStatusEntry(ctx, Config{}, pgConn, "PostgreSQL")
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, actual.ID)
	})
}
