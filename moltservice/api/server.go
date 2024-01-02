package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cockroachdb/molt/moltservice/gen/http/moltservice/server"
	"github.com/cockroachdb/molt/moltservice/gen/moltservice"
	"github.com/rs/zerolog"
	"goa.design/clue/metrics"
	goahttp "goa.design/goa/v3/http"
)

// ServerConfig contains all the information necessary to
// configure the service's server.
type ServerConfig struct {
	SourceURL         string
	TargetURL         string
	ListenPort        int
	MetricsListenPort int
	Logger            zerolog.Logger
	ShowDocsHTML      bool
	SkipMetrics       bool
	DebugMode         bool
}

type Server struct {
	// ServiceServer is the HTTP server for the application.
	ServiceServer *http.Server
	// HealthMetricsServer is the HTTP server for the health check and metrics.
	HealthMetricsServer *http.Server
}

func NewServer(ctx context.Context, cfg *ServerConfig) (*Server, error) {
	s, err := NewMOLTService(cfg)
	if err != nil {
		return nil, err
	}

	endpoints := moltservice.NewEndpoints(s)
	registerEndpointMiddlewares(s.logger, endpoints)

	mux := goahttp.NewMuxer()

	// Register and run the server.
	// The nil values tell the server to use the default values for error handlers,
	// formatters, and file system mounts.
	svr := server.New(endpoints, mux, goahttp.RequestDecoder, goahttp.ResponseEncoder,
		nil /* errHandler */, nil /* formatter */, nil, /* fileSystemGenHTTPOpenapiJSON */
		nil /* fileSystemAssetsDocsHTML */)

	pathDetails := getPathPatternDetails(svr.Mounts)

	ctx = metrics.Context(ctx, moltservice.ServiceName, metrics.WithRouteResolver(func(r *http.Request) string {
		res, err := findMatchingPattern(r.URL.Path, pathDetails)
		if err != nil {
			cfg.Logger.Err(err).Msg("failed to find matching route pattern")
		}

		if res == "" {
			res = r.URL.Path
		}

		return res
	}))
	var handler http.Handler = mux
	if !cfg.SkipMetrics {
		// Wraps the handler in the middleware for Prometheus metrics.
		handler = metrics.HTTP(ctx)(mux)
	}

	registerMiddlewares(ctx, cfg.Logger, svr)

	// If we are not supposed to show docs HTML, then substitute handler
	// with one that notes that the route is inaccessible.
	if !cfg.ShowDocsHTML {
		svr.AssetsDocsHTML = getNotFoundHandler(cfg.Logger)
	}

	server.Mount(mux, svr)

	return &Server{
		ServiceServer: &http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", cfg.ListenPort),
			Handler: handler,
		},
		HealthMetricsServer: &http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", cfg.MetricsListenPort),
			Handler: getHealthAndMetricsHandler(ctx),
		},
	}, nil
}

// getNotFoundHandler returns a handler that lets the client know a page could not be found.
// The use case of this is to hide routes conditionally.
func getNotFoundHandler(logger zerolog.Logger) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, err := w.Write([]byte("Route not accessible"))
		if err != nil {
			logger.Error().Err(err).Msg("failed to write response")
		}
	})
}
