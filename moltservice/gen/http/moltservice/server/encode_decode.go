// Code generated by goa v3.14.1, DO NOT EDIT.
//
// moltservice HTTP server encoders and decoders
//
// Command:
// $ goa gen github.com/cockroachdb/molt/moltservice/design -o ./moltservice

package server

import (
	"context"
	"io"
	"net/http"
	"strconv"

	moltservice "github.com/cockroachdb/molt/moltservice/gen/moltservice"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

// EncodeCreateFetchTaskResponse returns an encoder for responses returned by
// the moltservice create_fetch_task endpoint.
func EncodeCreateFetchTaskResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, any) error {
	return func(ctx context.Context, w http.ResponseWriter, v any) error {
		res, _ := v.(moltservice.FetchAttemptID)
		enc := encoder(ctx, w)
		body := res
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// DecodeCreateFetchTaskRequest returns a decoder for requests sent to the
// moltservice create_fetch_task endpoint.
func DecodeCreateFetchTaskRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body CreateFetchTaskRequestBody
			err  error
		)
		err = decoder(r).Decode(&body)
		if err != nil {
			if err == io.EOF {
				return nil, goa.MissingPayloadError()
			}
			return nil, goa.DecodePayloadError(err.Error())
		}
		err = ValidateCreateFetchTaskRequestBody(&body)
		if err != nil {
			return nil, err
		}
		payload := NewCreateFetchTaskCreateFetchPayload(&body)

		return payload, nil
	}
}

// EncodeGetFetchTasksResponse returns an encoder for responses returned by the
// moltservice get_fetch_tasks endpoint.
func EncodeGetFetchTasksResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, any) error {
	return func(ctx context.Context, w http.ResponseWriter, v any) error {
		res, _ := v.([]*moltservice.FetchRun)
		enc := encoder(ctx, w)
		body := NewGetFetchTasksResponseBody(res)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// EncodeGetSpecificFetchTaskResponse returns an encoder for responses returned
// by the moltservice get_specific_fetch_task endpoint.
func EncodeGetSpecificFetchTaskResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, any) error {
	return func(ctx context.Context, w http.ResponseWriter, v any) error {
		res, _ := v.(*moltservice.FetchRunDetailed)
		enc := encoder(ctx, w)
		body := NewGetSpecificFetchTaskResponseBody(res)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// DecodeGetSpecificFetchTaskRequest returns a decoder for requests sent to the
// moltservice get_specific_fetch_task endpoint.
func DecodeGetSpecificFetchTaskRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			id  int
			err error

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseInt(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "integer"))
			}
			id = int(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewGetSpecificFetchTaskPayload(id)

		return payload, nil
	}
}

// EncodeCreateVerifyTaskFromFetchResponse returns an encoder for responses
// returned by the moltservice create_verify_task_from_fetch endpoint.
func EncodeCreateVerifyTaskFromFetchResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, any) error {
	return func(ctx context.Context, w http.ResponseWriter, v any) error {
		res, _ := v.(moltservice.VerifyAttemptID)
		enc := encoder(ctx, w)
		body := res
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// DecodeCreateVerifyTaskFromFetchRequest returns a decoder for requests sent
// to the moltservice create_verify_task_from_fetch endpoint.
func DecodeCreateVerifyTaskFromFetchRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			id  int
			err error

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseInt(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "integer"))
			}
			id = int(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewCreateVerifyTaskFromFetchPayload(id)

		return payload, nil
	}
}

// EncodeGetVerifyTasksResponse returns an encoder for responses returned by
// the moltservice get_verify_tasks endpoint.
func EncodeGetVerifyTasksResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, any) error {
	return func(ctx context.Context, w http.ResponseWriter, v any) error {
		res, _ := v.([]*moltservice.VerifyRun)
		enc := encoder(ctx, w)
		body := NewGetVerifyTasksResponseBody(res)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// EncodeGetSpecificVerifyTaskResponse returns an encoder for responses
// returned by the moltservice get_specific_verify_task endpoint.
func EncodeGetSpecificVerifyTaskResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, any) error {
	return func(ctx context.Context, w http.ResponseWriter, v any) error {
		res, _ := v.(*moltservice.VerifyRunDetailed)
		enc := encoder(ctx, w)
		body := NewGetSpecificVerifyTaskResponseBody(res)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// DecodeGetSpecificVerifyTaskRequest returns a decoder for requests sent to
// the moltservice get_specific_verify_task endpoint.
func DecodeGetSpecificVerifyTaskRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			id  int
			err error

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseInt(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "integer"))
			}
			id = int(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewGetSpecificVerifyTaskPayload(id)

		return payload, nil
	}
}

// marshalMoltserviceFetchRunToFetchRunResponse builds a value of type
// *FetchRunResponse from a value of type *moltservice.FetchRun.
func marshalMoltserviceFetchRunToFetchRunResponse(v *moltservice.FetchRun) *FetchRunResponse {
	res := &FetchRunResponse{
		ID:         v.ID,
		Name:       v.Name,
		Status:     v.Status,
		StartedAt:  v.StartedAt,
		FinishedAt: v.FinishedAt,
	}

	return res
}

// marshalMoltserviceFetchStatsDetailedToFetchStatsDetailedResponseBody builds
// a value of type *FetchStatsDetailedResponseBody from a value of type
// *moltservice.FetchStatsDetailed.
func marshalMoltserviceFetchStatsDetailedToFetchStatsDetailedResponseBody(v *moltservice.FetchStatsDetailed) *FetchStatsDetailedResponseBody {
	if v == nil {
		return nil
	}
	res := &FetchStatsDetailedResponseBody{
		PercentComplete:  v.PercentComplete,
		NumErrors:        v.NumErrors,
		NumTables:        v.NumTables,
		NumRows:          v.NumRows,
		NetDurationMs:    v.NetDurationMs,
		ImportDurationMs: v.ImportDurationMs,
		ExportDurationMs: v.ExportDurationMs,
		CdcCursor:        v.CdcCursor,
	}

	return res
}

// marshalMoltserviceLogToLogResponseBody builds a value of type
// *LogResponseBody from a value of type *moltservice.Log.
func marshalMoltserviceLogToLogResponseBody(v *moltservice.Log) *LogResponseBody {
	res := &LogResponseBody{
		Timestamp: v.Timestamp,
		Level:     v.Level,
		Message:   v.Message,
	}

	return res
}

// marshalMoltserviceVerifyRunToVerifyRunResponseBody builds a value of type
// *VerifyRunResponseBody from a value of type *moltservice.VerifyRun.
func marshalMoltserviceVerifyRunToVerifyRunResponseBody(v *moltservice.VerifyRun) *VerifyRunResponseBody {
	res := &VerifyRunResponseBody{
		ID:         v.ID,
		Name:       v.Name,
		Status:     v.Status,
		StartedAt:  v.StartedAt,
		FinishedAt: v.FinishedAt,
		FetchID:    v.FetchID,
	}

	return res
}

// marshalMoltserviceVerifyRunToVerifyRunResponse builds a value of type
// *VerifyRunResponse from a value of type *moltservice.VerifyRun.
func marshalMoltserviceVerifyRunToVerifyRunResponse(v *moltservice.VerifyRun) *VerifyRunResponse {
	res := &VerifyRunResponse{
		ID:         v.ID,
		Name:       v.Name,
		Status:     v.Status,
		StartedAt:  v.StartedAt,
		FinishedAt: v.FinishedAt,
		FetchID:    v.FetchID,
	}

	return res
}

// marshalMoltserviceVerifyStatsDetailedToVerifyStatsDetailedResponseBody
// builds a value of type *VerifyStatsDetailedResponseBody from a value of type
// *moltservice.VerifyStatsDetailed.
func marshalMoltserviceVerifyStatsDetailedToVerifyStatsDetailedResponseBody(v *moltservice.VerifyStatsDetailed) *VerifyStatsDetailedResponseBody {
	res := &VerifyStatsDetailedResponseBody{
		NumTables:             v.NumTables,
		NumTruthRows:          v.NumTruthRows,
		NumSuccess:            v.NumSuccess,
		NumConditionalSuccess: v.NumConditionalSuccess,
		NumMissing:            v.NumMissing,
		NumMismatch:           v.NumMismatch,
		NumExtraneous:         v.NumExtraneous,
		NumLiveRetry:          v.NumLiveRetry,
		NumColumnMismatch:     v.NumColumnMismatch,
		NetDurationMs:         v.NetDurationMs,
	}

	return res
}

// marshalMoltserviceVerifyMismatchToVerifyMismatchResponseBody builds a value
// of type *VerifyMismatchResponseBody from a value of type
// *moltservice.VerifyMismatch.
func marshalMoltserviceVerifyMismatchToVerifyMismatchResponseBody(v *moltservice.VerifyMismatch) *VerifyMismatchResponseBody {
	res := &VerifyMismatchResponseBody{
		Timestamp: v.Timestamp,
		Level:     v.Level,
		Message:   v.Message,
		Schema:    v.Schema,
		Table:     v.Table,
		Type:      v.Type,
	}

	return res
}
