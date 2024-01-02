package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/cockroachdb/molt/moltservice/gen/http/moltservice/server"
	"github.com/cockroachdb/molt/moltservice/gen/moltservice"
	"github.com/rs/zerolog"
	"goa.design/clue/health"
	"goa.design/clue/metrics"
	"goa.design/goa/v3/http/middleware"
	goa "goa.design/goa/v3/pkg"
)

type CheckAPIVersionError struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// LogHandlerError is an endpoint middleware that logs handler errors using the
// given Logger.
func LogHandlerError(logger zerolog.Logger) func(goa.Endpoint) goa.Endpoint {
	return func(e goa.Endpoint) goa.Endpoint {
		// A Goa endpoint is itself a function.
		return goa.Endpoint(func(ctx context.Context, req interface{}) (interface{}, error) {
			// Call the original endpoint function.
			res, err := e(ctx, req)
			// Log any error.
			if err != nil {
				logger.Err(err).Msg("logging error from handler")
			}
			// Return endpoint results.
			return res, err
		})
	}
}

// LogHTTPDetails logs the HTTP request and response details
func LogHTTPDetails(logger zerolog.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			methodAndRoute := fmt.Sprintf("%s: %s", r.Method, r.URL.Path)
			clientIP := getClientIP(r)

			logger.Info().
				Str("client_ip", clientIP).
				Str("request_path", methodAndRoute).
				Msg("handling request")

			rw := middleware.CaptureResponse(w)
			h.ServeHTTP(rw, r)

			zlogEvent := logger.Info()
			if rw.StatusCode >= 400 {
				zlogEvent = logger.Error()
			}

			zlogEvent.
				Int("status_code", rw.StatusCode).
				Int("length_bytes", rw.ContentLength).
				Str("time", time.Since(started).String()).
				Str("request_path", methodAndRoute).
				Msg("completed request")

		})
	}
}

// getClientIP makes a best effort to compute the request client IP.
func getClientIP(req *http.Request) string {
	if f := req.Header.Get("X-Forwarded-For"); f != "" {
		return f
	}
	f := req.RemoteAddr
	ip, _, err := net.SplitHostPort(f)
	if err != nil {
		return f
	}
	return ip
}

// registerEndpointMiddlewares registers all middleware that interacts with the endpoint layer.
func registerEndpointMiddlewares(logger zerolog.Logger, endpoints *moltservice.Endpoints) {
	endpoints.Use(LogHandlerError(logger))
}

// registerMiddlewares registers all middleware that interacts with the transport (HTTP) layer.
func registerMiddlewares(ctx context.Context, logger zerolog.Logger, svr *server.Server) {
	svr.Use(LogHTTPDetails(logger))
}

// getHealthAndMetricsHandler returns a handler for
func getHealthAndMetricsHandler(ctx context.Context) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", metrics.Handler(ctx).ServeHTTP)
	// TODO: add dependency in NewChecker to the LMS whenever that is implemented.
	mux.HandleFunc("/healthz", health.Handler(health.NewChecker()))
	return mux
}
