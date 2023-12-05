package fetch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cockroachdb/datadriven"
	"github.com/cockroachdb/molt/compression"
	"github.com/cockroachdb/molt/dbconn"
	"github.com/cockroachdb/molt/fetch/datablobstorage"
	"github.com/cockroachdb/molt/fetch/dataexport"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/verify/dbverify"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

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
						return testutils.QueryConnCommand(t, d, conns)
					case "fetch":
						filter := dbverify.DefaultFilterConfig()
						truncate := true
						live := false
						direct := false
						compress := false

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

							localStoreListenAddr := ""
							localStoreCrdbAccessAddr := ""

							const darwinLocalhostEndpoint = "host.docker.internal"
							const linuxLocalhostEndpoint = "172.17.0.1"
							const localStorageServerPort = 4040

							// Resources:
							// https://stackoverflow.com/questions/48546124/what-is-linux-equivalent-of-host-docker-internal
							// https://docs.docker.com/desktop/networking/#i-want-to-connect-from-a-container-to-a-service-on-the-host
							// In the CI, the databases are all spin up in docker-compose,
							// which not necessarily share the network with the host.
							// When importing the data to the target database,
							// it requires the database reaches the local storage server
							// (spun up on host network) from the container (i.e. from
							// the container's network). According to the 2 links
							// above, the `localhost` on the host network is accessible
							// via different endpoint based on the operating system:
							// - Linux, Windows: 172.17.0.1
							// - MacOS: host.docker.internal
							switch runtime.GOOS {
							case "darwin":
								localStoreListenAddr = fmt.Sprintf("localhost:%d", localStorageServerPort)
								localStoreCrdbAccessAddr = fmt.Sprintf("%s:%d", darwinLocalhostEndpoint, localStorageServerPort)
							default:
								switch tc.desc {
								case "crdb":
									// Here the target db is cockroachdbtartget in .github/docker-compose.yaml,
									// which cannot be spun up on the host network (with ` network_mode: host`).
									// The reason is the docker image for crdb only allows listen-addr to be
									// localhost:26257, which will conflict with the `cockroachdb` container.
									// We thus has to let cockroachdbtartget lives in its own network and port-forward.
									// In this case in Linux, the host's localhost can only be accessed via `172.17.0.1`
									// from the container's network.
									localStoreListenAddr = fmt.Sprintf("%s:%d", linuxLocalhostEndpoint, localStorageServerPort)
								case "pg", "mysql":
									// Here the target db is cockroachdb in .github/docker-compose.yaml,
									// which is directly spun up on the host network (with ` network_mode: host`).
									// In Linux case, it can directly access the host server via localhost.
									localStoreListenAddr = fmt.Sprintf("localhost:%d", localStorageServerPort)
								}
								localStoreCrdbAccessAddr = localStoreListenAddr
							}

							src, err = datablobstorage.NewLocalStore(logger, dir, localStoreListenAddr, localStoreCrdbAccessAddr)
							require.NoError(t, err)
						}

						compressionFlag := compression.None
						if compress {
							compressionFlag = compression.GZIP
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
						)
						if expectError {
							require.Error(t, err)
							return err.Error()
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
