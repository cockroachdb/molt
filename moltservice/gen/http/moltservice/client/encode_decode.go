// Code generated by goa v3.14.1, DO NOT EDIT.
//
// moltservice HTTP client encoders and decoders
//
// Command:
// $ goa gen github.com/cockroachdb/molt/moltservice/design -o ./moltservice

package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	moltservice "github.com/cockroachdb/molt/moltservice/gen/moltservice"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

// BuildCreateFetchTaskRequest instantiates a HTTP request object with method
// and path set to call the "moltservice" service "create_fetch_task" endpoint
func (c *Client) BuildCreateFetchTaskRequest(ctx context.Context, v any) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: CreateFetchTaskMoltservicePath()}
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("moltservice", "create_fetch_task", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// EncodeCreateFetchTaskRequest returns an encoder for requests sent to the
// moltservice create_fetch_task server.
func EncodeCreateFetchTaskRequest(encoder func(*http.Request) goahttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*moltservice.CreateFetchPayload)
		if !ok {
			return goahttp.ErrInvalidType("moltservice", "create_fetch_task", "*moltservice.CreateFetchPayload", v)
		}
		body := NewCreateFetchTaskRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return goahttp.ErrEncodingError("moltservice", "create_fetch_task", err)
		}
		return nil
	}
}

// DecodeCreateFetchTaskResponse returns a decoder for responses returned by
// the moltservice create_fetch_task endpoint. restoreBody controls whether the
// response body should be restored after having been read.
func DecodeCreateFetchTaskResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body int
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("moltservice", "create_fetch_task", err)
			}
			res := NewCreateFetchTaskFetchAttemptIDOK(body)
			return res, nil
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("moltservice", "create_fetch_task", resp.StatusCode, string(body))
		}
	}
}

// BuildGetFetchTasksRequest instantiates a HTTP request object with method and
// path set to call the "moltservice" service "get_fetch_tasks" endpoint
func (c *Client) BuildGetFetchTasksRequest(ctx context.Context, v any) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: GetFetchTasksMoltservicePath()}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("moltservice", "get_fetch_tasks", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeGetFetchTasksResponse returns a decoder for responses returned by the
// moltservice get_fetch_tasks endpoint. restoreBody controls whether the
// response body should be restored after having been read.
func DecodeGetFetchTasksResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body GetFetchTasksResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("moltservice", "get_fetch_tasks", err)
			}
			for _, e := range body {
				if e != nil {
					if err2 := ValidateFetchRunResponse(e); err2 != nil {
						err = goa.MergeErrors(err, err2)
					}
				}
			}
			if err != nil {
				return nil, goahttp.ErrValidationError("moltservice", "get_fetch_tasks", err)
			}
			res := NewGetFetchTasksFetchRunOK(body)
			return res, nil
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("moltservice", "get_fetch_tasks", resp.StatusCode, string(body))
		}
	}
}

// BuildGetSpecificFetchTaskRequest instantiates a HTTP request object with
// method and path set to call the "moltservice" service
// "get_specific_fetch_task" endpoint
func (c *Client) BuildGetSpecificFetchTaskRequest(ctx context.Context, v any) (*http.Request, error) {
	var (
		id int
	)
	{
		p, ok := v.(*moltservice.GetSpecificFetchTaskPayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("moltservice", "get_specific_fetch_task", "*moltservice.GetSpecificFetchTaskPayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: GetSpecificFetchTaskMoltservicePath(id)}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("moltservice", "get_specific_fetch_task", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeGetSpecificFetchTaskResponse returns a decoder for responses returned
// by the moltservice get_specific_fetch_task endpoint. restoreBody controls
// whether the response body should be restored after having been read.
func DecodeGetSpecificFetchTaskResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body GetSpecificFetchTaskResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("moltservice", "get_specific_fetch_task", err)
			}
			err = ValidateGetSpecificFetchTaskResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("moltservice", "get_specific_fetch_task", err)
			}
			res := NewGetSpecificFetchTaskFetchRunDetailedOK(&body)
			return res, nil
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("moltservice", "get_specific_fetch_task", resp.StatusCode, string(body))
		}
	}
}

// unmarshalFetchRunResponseToMoltserviceFetchRun builds a value of type
// *moltservice.FetchRun from a value of type *FetchRunResponse.
func unmarshalFetchRunResponseToMoltserviceFetchRun(v *FetchRunResponse) *moltservice.FetchRun {
	res := &moltservice.FetchRun{
		ID:         *v.ID,
		Name:       *v.Name,
		Status:     *v.Status,
		StartedAt:  *v.StartedAt,
		FinishedAt: *v.FinishedAt,
	}

	return res
}

// unmarshalFetchStatsDetailedResponseBodyToMoltserviceFetchStatsDetailed
// builds a value of type *moltservice.FetchStatsDetailed from a value of type
// *FetchStatsDetailedResponseBody.
func unmarshalFetchStatsDetailedResponseBodyToMoltserviceFetchStatsDetailed(v *FetchStatsDetailedResponseBody) *moltservice.FetchStatsDetailed {
	if v == nil {
		return nil
	}
	res := &moltservice.FetchStatsDetailed{
		PercentComplete:  *v.PercentComplete,
		NumErrors:        *v.NumErrors,
		NumTables:        *v.NumTables,
		NumRows:          *v.NumRows,
		NetDurationMs:    *v.NetDurationMs,
		ImportDurationMs: *v.ImportDurationMs,
		ExportDurationMs: *v.ExportDurationMs,
		CdcCursor:        *v.CdcCursor,
	}

	return res
}

// unmarshalLogResponseBodyToMoltserviceLog builds a value of type
// *moltservice.Log from a value of type *LogResponseBody.
func unmarshalLogResponseBodyToMoltserviceLog(v *LogResponseBody) *moltservice.Log {
	res := &moltservice.Log{
		Timestamp: *v.Timestamp,
		Level:     *v.Level,
		Message:   *v.Message,
	}

	return res
}
