package datablobstorage

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbtable"
	"github.com/cockroachdb/molt/testutils"
	"github.com/cockroachdb/molt/utils"
	"github.com/rs/zerolog"
)

type localStore struct {
	logger         zerolog.Logger
	basePath       string
	cleanPaths     map[string]struct{}
	crdbAccessAddr string
	server         *http.Server
}

func NewLocalStore(
	logger zerolog.Logger, basePath string, listenAddr string, crdbAccessAddr string,
) (*localStore, error) {
	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		return nil, err
	}
	var server *http.Server
	if listenAddr != "" {
		if crdbAccessAddr == "" {
			ip := getLocalIP()
			if ip == "" {
				return nil, errors.Newf("cannot find IP")
			}
			splat := strings.Split(listenAddr, ":")
			if len(splat) < 2 {
				return nil, errors.Newf("listen addr must have port")
			}
			port := splat[1]
			crdbAccessAddr = ip + ":" + port
		}
		server = &http.Server{
			Addr:    listenAddr,
			Handler: http.FileServer(http.Dir(basePath)),
		}
		go func() {
			logger.Info().
				Str("listen-addr", listenAddr).
				Str("crdb-access-addr", crdbAccessAddr).
				Msgf("starting file server")
			if err := server.ListenAndServe(); err != nil && err == http.ErrServerClosed {
				logger.Info().Msgf("http server intentionally shut down")
			} else if err != nil {
				logger.Err(err).Msgf("error starting file server")
			}
		}()
	}
	return &localStore{
		logger:         logger,
		basePath:       basePath,
		crdbAccessAddr: crdbAccessAddr,
		server:         server,
	}, nil
}

// GetLocalIP returns the non loopback local IP of the host
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func (l *localStore) CreateFromReader(
	ctx context.Context,
	r io.Reader,
	table dbtable.VerifiedTable,
	iteration int,
	fileExt string,
	numRows chan int,
	testingKnobs testutils.FetchTestingKnobs,
) (Resource, error) {
	baseDir := path.Join(l.basePath, table.SafeString())
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		return nil, err
	}
	if testingKnobs.FailedWriteToBucket.FailedBeforeReadFromPipe {
		return nil, errors.New(LocalWriterMockErrMsg)
	}
	p := path.Join(baseDir, fmt.Sprintf("part_%08d.%s", iteration, fileExt))
	logger := l.logger.With().Str("path", p).Logger()
	logger.Debug().Msgf("creating file")
	f, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 1024*1024)
	rows := <-numRows

	if fileExt == "tar.gz" {
		// Need to create a GZIP writer so we can write GZIP data
		// to the file for the header.
		gf := gzip.NewWriter(f)
		fw := bufio.NewWriter(gf)
		if _, err := fw.WriteString(fmt.Sprintf("%d\n", rows)); err != nil {
			return nil, err
		}
		if err := fw.Flush(); err != nil {
			return nil, err
		}
		if err := gf.Close(); err != nil {
			return nil, err
		}
	} else {
		if _, err := f.WriteString(fmt.Sprintf("%d\n", rows)); err != nil {
			return nil, err
		}
	}

	for {
		n, err := r.Read(buf)
		if testingKnobs.FailedWriteToBucket.FailedAfterReadFromPipe {
			return nil, errors.New(LocalWriterMockErrMsg)
		}

		if err != nil {
			if err == io.EOF {
				logger.Debug().Msgf("wrote file")
				return &localResource{path: p, store: l, rows: rows}, nil
			}
			return nil, err
		}
		if _, err := f.Write(buf[:n]); err != nil {
			return nil, err
		}
	}
}

func (l *localStore) ListFromContinuationPoint(
	ctx context.Context, table dbtable.VerifiedTable, fileName string,
) ([]Resource, error) {
	baseDir := path.Join(l.basePath, table.SafeString())
	files, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	resources := []Resource{}
	for _, f := range files {
		if f.Name() >= fileName && utils.MatchesFileConvention(f.Name()) {
			p := path.Join(baseDir, f.Name())
			numRows, err := readFirstLine(p)
			if err != nil {
				l.logger.Err(err).Msg("failed to detect number of rows")
			}

			numRowsInt, err := strconv.Atoi(numRows)
			if err != nil {
				l.logger.Err(err).Msgf("failed to convert string %s to integer", numRows)
			}

			resources = append(resources, &localResource{
				path:  p,
				store: l,
				rows:  numRowsInt,
			})
		}

	}
	return resources, nil
}

func readFirstLine(filePath string) (string, error) {
	var reader io.Reader
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	reader = f

	// We need a gzip reader if we have a gzip extension.
	if filepath.Ext(filePath) == ".gz" {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gr.Close()
		reader = gr
	}

	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", nil
		}
		return "", fmt.Errorf("empty file")
	}
	return scanner.Text(), nil
}

func (l *localStore) DefaultFlushBatchSize() int {
	return 128 * 1024 * 1024
}

func (l *localStore) Cleanup(ctx context.Context) error {
	for p := range l.cleanPaths {
		if err := os.Remove(p); err != nil {
			return err
		}
	}

	if l.server != nil {
		return l.server.Shutdown(ctx)
	}

	return nil
}

func (l *localStore) CanBeTarget() bool {
	return true
}

func (l *localStore) TelemetryName() string {
	return "local"
}

type localResource struct {
	path  string
	store *localStore
	rows  int
}

func (l *localResource) Reader(ctx context.Context) (io.ReadCloser, error) {
	return os.Open(l.path)
}

func (l *localResource) ImportURL() (string, error) {
	if l.store.crdbAccessAddr == "" {
		return "", errors.AssertionFailedf("cannot IMPORT from a local path unless file server is set")
	}
	rel, err := filepath.Rel(l.store.basePath, l.path)
	if err != nil {
		return "", errors.Wrapf(err, "error finding relative path")
	}
	return fmt.Sprintf("http://%s/%s", l.store.crdbAccessAddr, rel), nil
}

func (l *localResource) Key() (string, error) {
	rel, err := filepath.Rel(l.store.basePath, l.path)
	if err != nil {
		return "", errors.Wrapf(err, "error finding relative path")
	}
	return rel, nil
}

func (l *localResource) Rows() int {
	return l.rows
}

func (l *localResource) MarkForCleanup(ctx context.Context) error {
	l.store.logger.Debug().Msgf("removing %s", l.path)
	return os.Remove(l.path)
}

func (l *localResource) IsLocal() bool {
	return true
}

const LocalWriterMockErrMsg = "forced error for local path storage"
