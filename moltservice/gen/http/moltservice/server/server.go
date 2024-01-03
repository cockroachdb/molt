// Code generated by goa v3.14.1, DO NOT EDIT.
//
// moltservice HTTP server
//
// Command:
// $ goa gen github.com/cockroachdb/molt/moltservice/design -o ./moltservice

package server

import (
	"context"
	"net/http"
	"os"

	moltservice "github.com/cockroachdb/molt/moltservice/gen/moltservice"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
	"goa.design/plugins/v3/cors"
)

// Server lists the moltservice service endpoint HTTP handlers.
type Server struct {
	Mounts               []*MountPoint
	CreateFetchTask      http.Handler
	GetFetchTasks        http.Handler
	GetSpecificFetchTask http.Handler
	CORS                 http.Handler
	GenHTTPOpenapiJSON   http.Handler
	AssetsDocsHTML       http.Handler
}

// MountPoint holds information about the mounted endpoints.
type MountPoint struct {
	// Method is the name of the service method served by the mounted HTTP handler.
	Method string
	// Verb is the HTTP method used to match requests to the mounted handler.
	Verb string
	// Pattern is the HTTP request path pattern used to match requests to the
	// mounted handler.
	Pattern string
}

// New instantiates HTTP handlers for all the moltservice service endpoints
// using the provided encoder and decoder. The handlers are mounted on the
// given mux using the HTTP verb and path defined in the design. errhandler is
// called whenever a response fails to be encoded. formatter is used to format
// errors returned by the service methods prior to encoding. Both errhandler
// and formatter are optional and can be nil.
func New(
	e *moltservice.Endpoints,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) goahttp.Statuser,
	fileSystemGenHTTPOpenapiJSON http.FileSystem,
	fileSystemAssetsDocsHTML http.FileSystem,
) *Server {
	if fileSystemGenHTTPOpenapiJSON == nil {
		fileSystemGenHTTPOpenapiJSON = http.Dir(".")
	}
	if fileSystemAssetsDocsHTML == nil {
		fileSystemAssetsDocsHTML = http.Dir(".")
	}
	return &Server{
		Mounts: []*MountPoint{
			{"CreateFetchTask", "POST", "/api/v1/fetch"},
			{"GetFetchTasks", "GET", "/api/v1/fetch"},
			{"GetSpecificFetchTask", "GET", "/api/v1/fetch/{id}"},
			{"CORS", "OPTIONS", "/api/v1/fetch"},
			{"CORS", "OPTIONS", "/api/v1/fetch/{id}"},
			{"CORS", "OPTIONS", "/openapi.json"},
			{"CORS", "OPTIONS", "/docs.html"},
			{"./gen/http/openapi.json", "GET", "/openapi.json"},
			{"./assets/docs.html", "GET", "/docs.html"},
		},
		CreateFetchTask:      NewCreateFetchTaskHandler(e.CreateFetchTask, mux, decoder, encoder, errhandler, formatter),
		GetFetchTasks:        NewGetFetchTasksHandler(e.GetFetchTasks, mux, decoder, encoder, errhandler, formatter),
		GetSpecificFetchTask: NewGetSpecificFetchTaskHandler(e.GetSpecificFetchTask, mux, decoder, encoder, errhandler, formatter),
		CORS:                 NewCORSHandler(),
		GenHTTPOpenapiJSON:   http.FileServer(fileSystemGenHTTPOpenapiJSON),
		AssetsDocsHTML:       http.FileServer(fileSystemAssetsDocsHTML),
	}
}

// Service returns the name of the service served.
func (s *Server) Service() string { return "moltservice" }

// Use wraps the server handlers with the given middleware.
func (s *Server) Use(m func(http.Handler) http.Handler) {
	s.CreateFetchTask = m(s.CreateFetchTask)
	s.GetFetchTasks = m(s.GetFetchTasks)
	s.GetSpecificFetchTask = m(s.GetSpecificFetchTask)
	s.CORS = m(s.CORS)
}

// MethodNames returns the methods served.
func (s *Server) MethodNames() []string { return moltservice.MethodNames[:] }

// Mount configures the mux to serve the moltservice endpoints.
func Mount(mux goahttp.Muxer, h *Server) {
	MountCreateFetchTaskHandler(mux, h.CreateFetchTask)
	MountGetFetchTasksHandler(mux, h.GetFetchTasks)
	MountGetSpecificFetchTaskHandler(mux, h.GetSpecificFetchTask)
	MountCORSHandler(mux, h.CORS)
	MountGenHTTPOpenapiJSON(mux, goahttp.Replace("", "/./gen/http/openapi.json", h.GenHTTPOpenapiJSON))
	MountAssetsDocsHTML(mux, goahttp.Replace("", "/./assets/docs.html", h.AssetsDocsHTML))
}

// Mount configures the mux to serve the moltservice endpoints.
func (s *Server) Mount(mux goahttp.Muxer) {
	Mount(mux, s)
}

// MountCreateFetchTaskHandler configures the mux to serve the "moltservice"
// service "create_fetch_task" endpoint.
func MountCreateFetchTaskHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleMoltserviceOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("POST", "/api/v1/fetch", f)
}

// NewCreateFetchTaskHandler creates a HTTP handler which loads the HTTP
// request and calls the "moltservice" service "create_fetch_task" endpoint.
func NewCreateFetchTaskHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeCreateFetchTaskRequest(mux, decoder)
		encodeResponse = EncodeCreateFetchTaskResponse(encoder)
		encodeError    = goahttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "create_fetch_task")
		ctx = context.WithValue(ctx, goa.ServiceKey, "moltservice")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountGetFetchTasksHandler configures the mux to serve the "moltservice"
// service "get_fetch_tasks" endpoint.
func MountGetFetchTasksHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleMoltserviceOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("GET", "/api/v1/fetch", f)
}

// NewGetFetchTasksHandler creates a HTTP handler which loads the HTTP request
// and calls the "moltservice" service "get_fetch_tasks" endpoint.
func NewGetFetchTasksHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) goahttp.Statuser,
) http.Handler {
	var (
		encodeResponse = EncodeGetFetchTasksResponse(encoder)
		encodeError    = goahttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "get_fetch_tasks")
		ctx = context.WithValue(ctx, goa.ServiceKey, "moltservice")
		var err error
		res, err := endpoint(ctx, nil)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountGetSpecificFetchTaskHandler configures the mux to serve the
// "moltservice" service "get_specific_fetch_task" endpoint.
func MountGetSpecificFetchTaskHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleMoltserviceOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("GET", "/api/v1/fetch/{id}", f)
}

// NewGetSpecificFetchTaskHandler creates a HTTP handler which loads the HTTP
// request and calls the "moltservice" service "get_specific_fetch_task"
// endpoint.
func NewGetSpecificFetchTaskHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeGetSpecificFetchTaskRequest(mux, decoder)
		encodeResponse = EncodeGetSpecificFetchTaskResponse(encoder)
		encodeError    = goahttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "get_specific_fetch_task")
		ctx = context.WithValue(ctx, goa.ServiceKey, "moltservice")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountGenHTTPOpenapiJSON configures the mux to serve GET request made to
// "/openapi.json".
func MountGenHTTPOpenapiJSON(mux goahttp.Muxer, h http.Handler) {
	mux.Handle("GET", "/openapi.json", HandleMoltserviceOrigin(h).ServeHTTP)
}

// MountAssetsDocsHTML configures the mux to serve GET request made to
// "/docs.html".
func MountAssetsDocsHTML(mux goahttp.Muxer, h http.Handler) {
	mux.Handle("GET", "/docs.html", HandleMoltserviceOrigin(h).ServeHTTP)
}

// MountCORSHandler configures the mux to serve the CORS endpoints for the
// service moltservice.
func MountCORSHandler(mux goahttp.Muxer, h http.Handler) {
	h = HandleMoltserviceOrigin(h)
	mux.Handle("OPTIONS", "/api/v1/fetch", h.ServeHTTP)
	mux.Handle("OPTIONS", "/api/v1/fetch/{id}", h.ServeHTTP)
	mux.Handle("OPTIONS", "/openapi.json", h.ServeHTTP)
	mux.Handle("OPTIONS", "/docs.html", h.ServeHTTP)
}

// NewCORSHandler creates a HTTP handler which returns a simple 204 response.
func NewCORSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
}

// HandleMoltserviceOrigin applies the CORS response headers corresponding to
// the origin for the service moltservice.
func HandleMoltserviceOrigin(h http.Handler) http.Handler {
	originStr0, present := os.LookupEnv("MOLT_SERVICE_ALLOW_ORIGIN")
	if !present {
		panic("CORS origin environment variable \"MOLT_SERVICE_ALLOW_ORIGIN\" not set!")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Not a CORS request
			h.ServeHTTP(w, r)
			return
		}
		if cors.MatchOrigin(origin, originStr0) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if acrm := r.Header.Get("Access-Control-Request-Method"); acrm != "" {
				// We are handling a preflight request
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.WriteHeader(204)
				return
			}
			h.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
		return
	})
}
