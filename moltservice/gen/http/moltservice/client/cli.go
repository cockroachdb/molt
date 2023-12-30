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
			return nil, fmt.Errorf("invalid JSON for body, \nerror: %s, \nexample of valid JSON:\n%s", err, "'{\n      \"bucket_name\": \"http://localhost:5000\",\n      \"bucket_path\": \"fetch/export\",\n      \"cleanup_intermediary_store\": true,\n      \"compression\": \"none\",\n      \"local_path\": \"/usr/Documents/fetch\",\n      \"local_path_crdb_address\": \"http://localhost:5000\",\n      \"local_path_listen_address\": \"http://localhost:5000\",\n      \"log_file\": \"task.log\",\n      \"mode\": \"DIRECT_COPY\",\n      \"num_batch_rows_export\": 100000,\n      \"num_concurrent_tables\": 4,\n      \"num_flush_bytes\": 2000,\n      \"num_flush_rows\": 200000,\n      \"pg_drop_slot\": false,\n      \"pg_logical_plugin\": \"my_plugin\",\n      \"pg_logical_slot_name\": \"my_slot\",\n      \"source_conn\": \"postgres://postgres:postgres@localhost:5432/molt?sslmode=disable\",\n      \"store\": \"Local\",\n      \"target_conn\": \"postgres://root@localhost:26257/defaultdb?sslmode=disable\",\n      \"truncate\": true\n   }'")
		}
		if !(body.Mode == "IMPORT_INTO" || body.Mode == "COPY_FROM" || body.Mode == "DIRECT_COPY") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.mode", body.Mode, []any{"IMPORT_INTO", "COPY_FROM", "DIRECT_COPY"}))
		}
		if !(body.Store == "None" || body.Store == "AWS" || body.Store == "GCP" || body.Store == "Local") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.store", body.Store, []any{"None", "AWS", "GCP", "Local"}))
		}
		if !(body.Compression == "gzip" || body.Compression == "none") {
			err = goa.MergeErrors(err, goa.InvalidEnumValueError("body.compression", body.Compression, []any{"gzip", "none"}))
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
	}

	return v, nil
}
