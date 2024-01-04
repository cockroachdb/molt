// Code generated by goa v3.14.1, DO NOT EDIT.
//
// moltservice HTTP server types
//
// Command:
// $ goa gen github.com/cockroachdb/molt/moltservice/design -o ./moltservice

package server

import (
	moltservice "github.com/cockroachdb/molt/moltservice/gen/moltservice"
	goa "goa.design/goa/v3/pkg"
)

// CreateFetchTaskRequestBody is the type of the "moltservice" service
// "create_fetch_task" endpoint HTTP request body.
type CreateFetchTaskRequestBody struct {
	// source database connection string
	SourceConn *string `form:"source_conn,omitempty" json:"source_conn,omitempty" xml:"source_conn,omitempty"`
	// target database connection string
	TargetConn *string `form:"target_conn,omitempty" json:"target_conn,omitempty" xml:"target_conn,omitempty"`
	// Mode of operation for fetch
	Mode *string `form:"mode,omitempty" json:"mode,omitempty" xml:"mode,omitempty"`
	// Type of intermediary store
	Store *string `form:"store,omitempty" json:"store,omitempty" xml:"store,omitempty"`
	// whether the intermediate store should be cleaned up after the fetch task
	CleanupIntermediaryStore *bool `form:"cleanup_intermediary_store,omitempty" json:"cleanup_intermediary_store,omitempty" xml:"cleanup_intermediary_store,omitempty"`
	// the absolute or relative path to write export files
	LocalPath *string `form:"local_path,omitempty" json:"local_path,omitempty" xml:"local_path,omitempty"`
	// the local address where the file server will be spun up
	LocalPathListenAddress *string `form:"local_path_listen_address,omitempty" json:"local_path_listen_address,omitempty" xml:"local_path_listen_address,omitempty"`
	// the local address CRDB will use to access the import files
	LocalPathCrdbAddress *string `form:"local_path_crdb_address,omitempty" json:"local_path_crdb_address,omitempty" xml:"local_path_crdb_address,omitempty"`
	// the local address CRDB will use to access the import files
	BucketName *string `form:"bucket_name,omitempty" json:"bucket_name,omitempty" xml:"bucket_name,omitempty"`
	// the sub-path within the bucket to write the export files
	BucketPath *string `form:"bucket_path,omitempty" json:"bucket_path,omitempty" xml:"bucket_path,omitempty"`
	// if specified, writes task execution logs to a file and stdout; otherwise,
	// just writes to stdout
	LogFile *string `form:"log_file,omitempty" json:"log_file,omitempty" xml:"log_file,omitempty"`
	// if specified, truncates the target tables before running the data load
	Truncate *bool `form:"truncate,omitempty" json:"truncate,omitempty" xml:"truncate,omitempty"`
	// compression type
	Compression *string `form:"compression,omitempty" json:"compression,omitempty" xml:"compression,omitempty"`
	// number of rows for the export before data is flushed to the disk (persisted)
	NumFlushRows *int `form:"num_flush_rows,omitempty" json:"num_flush_rows,omitempty" xml:"num_flush_rows,omitempty"`
	// number of bytes for the export before data is flushed to the disk
	NumFlushBytes *int `form:"num_flush_bytes,omitempty" json:"num_flush_bytes,omitempty" xml:"num_flush_bytes,omitempty"`
	// number of tables to process at the same time; this is usually sized based on
	// number of CPUs
	NumConcurrentTables *int `form:"num_concurrent_tables,omitempty" json:"num_concurrent_tables,omitempty" xml:"num_concurrent_tables,omitempty"`
	// number of rows to export at a given time for each iteration; tune this so
	// that you get most out of CPU and can batch the most data together
	NumBatchRowsExport *int `form:"num_batch_rows_export,omitempty" json:"num_batch_rows_export,omitempty" xml:"num_batch_rows_export,omitempty"`
	// name for pg replication slot
	PgLogicalSlotName *string `form:"pg_logical_slot_name,omitempty" json:"pg_logical_slot_name,omitempty" xml:"pg_logical_slot_name,omitempty"`
	// name for pg replication plugin
	PgLogicalPlugin *string `form:"pg_logical_plugin,omitempty" json:"pg_logical_plugin,omitempty" xml:"pg_logical_plugin,omitempty"`
	// if set and exists, drops the existing replication slot
	PgDropSlot *bool `form:"pg_drop_slot,omitempty" json:"pg_drop_slot,omitempty" xml:"pg_drop_slot,omitempty"`
	// the name of the fetch run
	Name *string `form:"name,omitempty" json:"name,omitempty" xml:"name,omitempty"`
}

// GetFetchTasksResponseBody is the type of the "moltservice" service
// "get_fetch_tasks" endpoint HTTP response body.
type GetFetchTasksResponseBody []*FetchRunResponse

// GetSpecificFetchTaskResponseBody is the type of the "moltservice" service
// "get_specific_fetch_task" endpoint HTTP response body.
type GetSpecificFetchTaskResponseBody struct {
	// ID of the run
	ID int `form:"id" json:"id" xml:"id"`
	// name of the fetch run
	Name string `form:"name" json:"name" xml:"name"`
	// status of the fetch run
	Status string `form:"status" json:"status" xml:"status"`
	// started at time
	StartedAt int `form:"started_at" json:"started_at" xml:"started_at"`
	// finished at time
	FinishedAt int `form:"finished_at" json:"finished_at" xml:"finished_at"`
	// fetch statistics
	Stats *FetchStatsDetailedResponseBody `form:"stats,omitempty" json:"stats,omitempty" xml:"stats,omitempty"`
	// logs for fetch run
	Logs []*LogResponseBody `form:"logs" json:"logs" xml:"logs"`
}

// FetchRunResponse is used to define fields on response body types.
type FetchRunResponse struct {
	// ID of the run
	ID int `form:"id" json:"id" xml:"id"`
	// name of the fetch run
	Name string `form:"name" json:"name" xml:"name"`
	// status of the fetch run
	Status string `form:"status" json:"status" xml:"status"`
	// started at time
	StartedAt int `form:"started_at" json:"started_at" xml:"started_at"`
	// finished at time
	FinishedAt int `form:"finished_at" json:"finished_at" xml:"finished_at"`
}

// FetchStatsDetailedResponseBody is used to define fields on response body
// types.
type FetchStatsDetailedResponseBody struct {
	// percentage complete of fetch run
	PercentComplete string `form:"percent_complete" json:"percent_complete" xml:"percent_complete"`
	// number of errors processed
	NumErrors int `form:"num_errors" json:"num_errors" xml:"num_errors"`
	// number of tables processed
	NumTables int `form:"num_tables" json:"num_tables" xml:"num_tables"`
	// number of rows
	NumRows int `form:"num_rows" json:"num_rows" xml:"num_rows"`
	// net duration in milliseconds
	NetDurationMs float64 `form:"net_duration_ms" json:"net_duration_ms" xml:"net_duration_ms"`
	// import duration in milliseconds
	ImportDurationMs float64 `form:"import_duration_ms" json:"import_duration_ms" xml:"import_duration_ms"`
	// export duration in milliseconds
	ExportDurationMs float64 `form:"export_duration_ms" json:"export_duration_ms" xml:"export_duration_ms"`
	// CDC cursor
	CdcCursor string `form:"cdc_cursor" json:"cdc_cursor" xml:"cdc_cursor"`
}

// LogResponseBody is used to define fields on response body types.
type LogResponseBody struct {
	// timestamp of log
	Timestamp int `form:"timestamp" json:"timestamp" xml:"timestamp"`
	// level for logging
	Level string `form:"level" json:"level" xml:"level"`
	// message for the logging
	Message string `form:"message" json:"message" xml:"message"`
}

// NewGetFetchTasksResponseBody builds the HTTP response body from the result
// of the "get_fetch_tasks" endpoint of the "moltservice" service.
func NewGetFetchTasksResponseBody(res []*moltservice.FetchRun) GetFetchTasksResponseBody {
	body := make([]*FetchRunResponse, len(res))
	for i, val := range res {
		body[i] = marshalMoltserviceFetchRunToFetchRunResponse(val)
	}
	return body
}

// NewGetSpecificFetchTaskResponseBody builds the HTTP response body from the
// result of the "get_specific_fetch_task" endpoint of the "moltservice"
// service.
func NewGetSpecificFetchTaskResponseBody(res *moltservice.FetchRunDetailed) *GetSpecificFetchTaskResponseBody {
	body := &GetSpecificFetchTaskResponseBody{
		ID:         res.ID,
		Name:       res.Name,
		Status:     res.Status,
		StartedAt:  res.StartedAt,
		FinishedAt: res.FinishedAt,
	}
	if res.Stats != nil {
		body.Stats = marshalMoltserviceFetchStatsDetailedToFetchStatsDetailedResponseBody(res.Stats)
	}
	if res.Logs != nil {
		body.Logs = make([]*LogResponseBody, len(res.Logs))
		for i, val := range res.Logs {
			body.Logs[i] = marshalMoltserviceLogToLogResponseBody(val)
		}
	} else {
		body.Logs = []*LogResponseBody{}
	}
	return body
}

// NewCreateFetchTaskCreateFetchPayload builds a moltservice service
// create_fetch_task endpoint payload.
func NewCreateFetchTaskCreateFetchPayload(body *CreateFetchTaskRequestBody) *moltservice.CreateFetchPayload {
	v := &moltservice.CreateFetchPayload{
		SourceConn:               *body.SourceConn,
		TargetConn:               *body.TargetConn,
		Mode:                     *body.Mode,
		Store:                    *body.Store,
		CleanupIntermediaryStore: *body.CleanupIntermediaryStore,
		LocalPath:                *body.LocalPath,
		LocalPathListenAddress:   *body.LocalPathListenAddress,
		LocalPathCrdbAddress:     *body.LocalPathCrdbAddress,
		BucketName:               *body.BucketName,
		BucketPath:               *body.BucketPath,
		LogFile:                  *body.LogFile,
		Truncate:                 *body.Truncate,
		Compression:              *body.Compression,
		NumFlushRows:             *body.NumFlushRows,
		NumFlushBytes:            *body.NumFlushBytes,
		NumConcurrentTables:      *body.NumConcurrentTables,
		NumBatchRowsExport:       *body.NumBatchRowsExport,
		PgLogicalSlotName:        *body.PgLogicalSlotName,
		PgLogicalPlugin:          *body.PgLogicalPlugin,
		PgDropSlot:               *body.PgDropSlot,
		Name:                     *body.Name,
	}

	return v
}

// NewGetSpecificFetchTaskPayload builds a moltservice service
// get_specific_fetch_task endpoint payload.
func NewGetSpecificFetchTaskPayload(id int) *moltservice.GetSpecificFetchTaskPayload {
	v := &moltservice.GetSpecificFetchTaskPayload{}
	v.ID = id

	return v
}

// ValidateCreateFetchTaskRequestBody runs the validations defined on
// create_fetch_task_request_body
func ValidateCreateFetchTaskRequestBody(body *CreateFetchTaskRequestBody) (err error) {
	if body.SourceConn == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("source_conn", "body"))
	}
	if body.TargetConn == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("target_conn", "body"))
	}
	if body.Mode == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("mode", "body"))
	}
	if body.Store == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("store", "body"))
	}
	if body.CleanupIntermediaryStore == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("cleanup_intermediary_store", "body"))
	}
	if body.LocalPath == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("local_path", "body"))
	}
	if body.LocalPathListenAddress == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("local_path_listen_address", "body"))
	}
	if body.LocalPathCrdbAddress == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("local_path_crdb_address", "body"))
	}
	if body.BucketName == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("bucket_name", "body"))
	}
	if body.BucketPath == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("bucket_path", "body"))
	}
	if body.LogFile == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("log_file", "body"))
	}
	if body.Truncate == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("truncate", "body"))
	}
	if body.Compression == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("compression", "body"))
	}
	if body.NumFlushRows == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("num_flush_rows", "body"))
	}
	if body.NumFlushBytes == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("num_flush_bytes", "body"))
	}
	if body.NumConcurrentTables == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("num_concurrent_tables", "body"))
	}
	if body.NumBatchRowsExport == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("num_batch_rows_export", "body"))
	}
	if body.PgLogicalSlotName == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("pg_logical_slot_name", "body"))
	}
	if body.PgLogicalPlugin == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("pg_logical_plugin", "body"))
	}
	if body.PgDropSlot == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("pg_drop_slot", "body"))
	}
	if body.Name == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("name", "body"))
	}
	if body.Mode != nil {
		if !(*body.Mode == "IMPORT_INTO" || *body.Mode == "COPY_FROM" || *body.Mode == "DIRECT_COPY") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.mode", *body.Mode, []any{"IMPORT_INTO", "COPY_FROM", "DIRECT_COPY"}))
		}
	}
	if body.Store != nil {
		if !(*body.Store == "None" || *body.Store == "AWS" || *body.Store == "GCP" || *body.Store == "Local") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.store", *body.Store, []any{"None", "AWS", "GCP", "Local"}))
		}
	}
	if body.Compression != nil {
		if !(*body.Compression == "gzip" || *body.Compression == "none" || *body.Compression == "default") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.compression", *body.Compression, []any{"gzip", "none", "default"}))
		}
	}
	return
}
