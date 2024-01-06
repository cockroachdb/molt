// Code generated by goa v3.14.1, DO NOT EDIT.
//
// moltservice HTTP client CLI support package
//
// Command:
// $ goa gen github.com/cockroachdb/molt/moltservice/design -o ./moltservice

package client

import (
	"encoding/json"
	"fmt"
	"strconv"

	moltservice "github.com/cockroachdb/molt/moltservice/gen/moltservice"
	goa "goa.design/goa/v3/pkg"
)

// BuildCreateFetchTaskPayload builds the payload for the moltservice
// create_fetch_task endpoint from CLI flags.
func BuildCreateFetchTaskPayload(moltserviceCreateFetchTaskBody string) (*moltservice.CreateFetchPayload, error) {
	var err error
	var body CreateFetchTaskRequestBody
	{
		err = json.Unmarshal([]byte(moltserviceCreateFetchTaskBody), &body)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON for body, \nerror: %s, \nexample of valid JSON:\n%s", err, "'{\n      \"bucket_name\": \"http://localhost:5000\",\n      \"bucket_path\": \"fetch/export\",\n      \"cleanup_intermediary_store\": true,\n      \"compression\": \"none\",\n      \"local_path\": \"/usr/Documents/fetch\",\n      \"local_path_crdb_address\": \"http://localhost:5000\",\n      \"local_path_listen_address\": \"http://localhost:5000\",\n      \"log_file\": \"task.log\",\n      \"mode\": \"DIRECT_COPY\",\n      \"name\": \"rluu pg to cockroach\",\n      \"num_batch_rows_export\": 100000,\n      \"num_concurrent_tables\": 4,\n      \"num_flush_bytes\": 2000000,\n      \"num_flush_rows\": 200000,\n      \"pg_drop_slot\": false,\n      \"pg_logical_plugin\": \"\",\n      \"pg_logical_slot_name\": \"\",\n      \"source_conn\": \"postgres://postgres:postgres@localhost:5432/molt?sslmode=disable\",\n      \"store\": \"Local\",\n      \"target_conn\": \"postgres://root@localhost:26257/defaultdb?sslmode=disable\",\n      \"truncate\": true\n   }'")
		}
		if !(body.Mode == "IMPORT_INTO" || body.Mode == "COPY_FROM" || body.Mode == "DIRECT_COPY") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.mode", body.Mode, []any{"IMPORT_INTO", "COPY_FROM", "DIRECT_COPY"}))
		}
		if !(body.Store == "None" || body.Store == "AWS" || body.Store == "GCP" || body.Store == "Local") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.store", body.Store, []any{"None", "AWS", "GCP", "Local"}))
		}
		if !(body.Compression == "gzip" || body.Compression == "none" || body.Compression == "default") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.compression", body.Compression, []any{"gzip", "none", "default"}))
		}
		if err != nil {
			return nil, err
		}
	}
	v := &moltservice.CreateFetchPayload{
		SourceConn:               body.SourceConn,
		TargetConn:               body.TargetConn,
		Mode:                     body.Mode,
		Store:                    body.Store,
		CleanupIntermediaryStore: body.CleanupIntermediaryStore,
		LocalPath:                body.LocalPath,
		LocalPathListenAddress:   body.LocalPathListenAddress,
		LocalPathCrdbAddress:     body.LocalPathCrdbAddress,
		BucketName:               body.BucketName,
		BucketPath:               body.BucketPath,
		LogFile:                  body.LogFile,
		Truncate:                 body.Truncate,
		Compression:              body.Compression,
		NumFlushRows:             body.NumFlushRows,
		NumFlushBytes:            body.NumFlushBytes,
		NumConcurrentTables:      body.NumConcurrentTables,
		NumBatchRowsExport:       body.NumBatchRowsExport,
		PgLogicalSlotName:        body.PgLogicalSlotName,
		PgLogicalPlugin:          body.PgLogicalPlugin,
		PgDropSlot:               body.PgDropSlot,
		Name:                     body.Name,
	}

	return v, nil
}

// BuildGetSpecificFetchTaskPayload builds the payload for the moltservice
// get_specific_fetch_task endpoint from CLI flags.
func BuildGetSpecificFetchTaskPayload(moltserviceGetSpecificFetchTaskID string) (*moltservice.GetSpecificFetchTaskPayload, error) {
	var err error
	var id int
	{
		var v int64
		v, err = strconv.ParseInt(moltserviceGetSpecificFetchTaskID, 10, strconv.IntSize)
		id = int(v)
		if err != nil {
			return nil, fmt.Errorf("invalid value for id, must be INT")
		}
	}
	v := &moltservice.GetSpecificFetchTaskPayload{}
	v.ID = id

	return v, nil
}

// BuildCreateVerifyTaskFromFetchPayload builds the payload for the moltservice
// create_verify_task_from_fetch endpoint from CLI flags.
func BuildCreateVerifyTaskFromFetchPayload(moltserviceCreateVerifyTaskFromFetchID string) (*moltservice.CreateVerifyTaskFromFetchPayload, error) {
	var err error
	var id int
	{
		var v int64
		v, err = strconv.ParseInt(moltserviceCreateVerifyTaskFromFetchID, 10, strconv.IntSize)
		id = int(v)
		if err != nil {
			return nil, fmt.Errorf("invalid value for id, must be INT")
		}
	}
	v := &moltservice.CreateVerifyTaskFromFetchPayload{}
	v.ID = id

	return v, nil
}

// BuildGetSpecificVerifyTaskPayload builds the payload for the moltservice
// get_specific_verify_task endpoint from CLI flags.
func BuildGetSpecificVerifyTaskPayload(moltserviceGetSpecificVerifyTaskID string) (*moltservice.GetSpecificVerifyTaskPayload, error) {
	var err error
	var id int
	{
		var v int64
		v, err = strconv.ParseInt(moltserviceGetSpecificVerifyTaskID, 10, strconv.IntSize)
		id = int(v)
		if err != nil {
			return nil, fmt.Errorf("invalid value for id, must be INT")
		}
	}
	v := &moltservice.GetSpecificVerifyTaskPayload{}
	v.ID = id

	return v, nil
}
