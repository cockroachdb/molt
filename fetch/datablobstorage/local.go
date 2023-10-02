package datablobstorage

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/dbtable"
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
			if err := server.ListenAndServe(); err != nil {
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
	ctx context.Context, r io.Reader, table dbtable.VerifiedTable, iteration int, fileExt string,
) (Resource, error) {
	baseDir := path.Join(l.basePath, table.SafeString())
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		return nil, err
	}
	p := path.Join(baseDir, fmt.Sprintf("part_%08d.%s", iteration, fileExt))
	logger := l.logger.With().Str("path", p).Logger()
	logger.Debug().Msgf("creating file")
	f, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 1024*1024)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				logger.Debug().Msgf("wrote file")
				return &localResource{path: p, store: l}, nil
			}
			return nil, err
		}
		if _, err := f.Write(buf[:n]); err != nil {
			return nil, err
		}
	}
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

func (l *localResource) MarkForCleanup(ctx context.Context) error {
	l.store.logger.Debug().Msgf("removing %s", l.path)
	return os.Remove(l.path)
}
