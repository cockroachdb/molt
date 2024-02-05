package fetch

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/cockroachdb/datadriven"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/dataexport"
	"github.com/cockroachdb/molt/fetch/status"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/verify/dbverify"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

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

				datadriven.RunTest(t, path, func(t *testing.T, d *datadriven.TestData) string {
					// Extract common arguments.
					args := d.CmdArgs[:0]
					var expectError bool
					for _, arg := range d.CmdArgs {
						switch arg.Key {
						case "expect-error":
							expectError = true
						default:
							args = append(args, arg)
						}
					}
					d.CmdArgs = args

					switch d.Cmd {
					case "exec":
						return testutils.ExecConnTestdata(t, d, conns)
					case "query":
						return replaceDockerInternalLocalHost(testutils.QueryConnCommand(t, d, conns))
					case "fetch":
						filter := dbverify.DefaultFilterConfig()
						truncate := true
						live := false
						direct := false
						compress := false
						corruptCSVFile := false

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
							default:
								t.Errorf("unknown key %s", cmd.Key)
							}
						}
						dir, err := os.MkdirTemp("", "")
						require.NoError(t, err)
						var src datablobstorage.Store
						defer func() {
							if src != nil {
								require.NoError(t, src.Cleanup(ctx))
							}
						}()
						if direct {
							src = datablobstorage.NewCopyCRDBDirect(logger, conns[1].(*dbconn.PGConn).Conn)
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

						knobs := testutils.FetchTestingKnobs{}
						if corruptCSVFile {
							knobs.TriggerCorruptCSVFile = true
						}

						err = Fetch(
							ctx,
							Config{
								Live:     live,
								Truncate: truncate,
								ExportSettings: dataexport.Settings{
									RowBatchSize: 2,
								},
								Compression: compressionFlag,
							},
							logger,
							conns,
							src,
							filter,
							knobs,
						)
						if expectError {
							require.Error(t, err)
							return replaceDockerInternalLocalHost(err.Error())
						}
						require.NoError(t, err)
						return ""
					default:
						t.Errorf("unknown command: %s", d.Cmd)
					}

					return ""
				})
			})
		})
	}
}

func TestInitStatusEntry(t *testing.T) {
	ctx := context.Background()
	dbName := "fetch_test_status"

	t.Run("successfully initialized when tables not created", func(t *testing.T) {
		conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
		require.NoError(t, err)
		pgConn := conn.(*dbconn.PGConn).Conn

		actual, err := initStatusEntry(ctx, pgConn, "PostgreSQL")
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, actual.ID)
	})

	t.Run("successfully initialized when tables created beforehand", func(t *testing.T) {
		conn, err := dbconn.TestOnlyCleanDatabase(ctx, "target", testutils.CRDBConnStr(), dbName)
		require.NoError(t, err)
		pgConn := conn.(*dbconn.PGConn).Conn
		// Setup the tables that we need to write for status.
		require.NoError(t, status.CreateStatusAndExceptionTables(ctx, pgConn))

		actual, err := initStatusEntry(ctx, pgConn, "PostgreSQL")
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, actual.ID)
	})
}
