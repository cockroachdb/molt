// Copyright 2023 Cockroach Labs Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package design

import . "goa.design/goa/v3/dsl"

var CreateFetchPayload = Type("create_fetch_payload", func() {
	Field(1, "source_conn", String, "source database connection string", func() {
		Example("postgres://postgres:postgres@localhost:5432/molt?sslmode=disable")
	})

	Field(2, "target_conn", String, "target database connection string", func() {
		Example("postgres://root@localhost:26257/defaultdb?sslmode=disable")
	})

	Field(3, "mode", String, "Mode of operation for fetch",
		func() {
			Enum("IMPORT_INTO", "COPY_FROM", "DIRECT_COPY")
			Example("IMPORT_INTO")
			Example("COPY_FROM")
			Example("DIRECT_COPY")
		},
	)

	Field(4, "store", String, "Type of intermediary store",
		func() {
			Enum("None", "AWS", "GCP", "Local")
			Example("None")
			Example("AWS")
			Example("GCP")
			Example("Local")
		},
	)

	Field(5, "cleanup_intermediary_store", Boolean, "whether the intermediate store should be cleaned up after the fetch task",
		func() {
			Example(true)
		},
	)

	Field(6, "local_path", String, "the absolute or relative path to write export files",
		func() {
			Example("/usr/Documents/fetch")
		},
	)

	Field(7, "local_path_listen_address", String, "the local address where the file server will be spun up",
		func() {
			Example("http://localhost:5000")
		},
	)

	Field(8, "local_path_crdb_address", String, "the local address CRDB will use to access the import files",
		func() {
			Example("http://localhost:5000")
		},
	)

	Field(9, "bucket_name", String, "the local address CRDB will use to access the import files",
		func() {
			Example("http://localhost:5000")
		},
	)

	Field(10, "bucket_path", String, "the sub-path within the bucket to write the export files",
		func() {
			Example("fetch/export")
		},
	)

	Field(11, "log_file", String, "if specified, writes task execution logs to a file and stdout; otherwise, just writes to stdout",
		func() {
			Example("task.log")
		},
	)

	Field(12, "truncate", Boolean, "if specified, truncates the target tables before running the data load",
		func() {
			Example(true)
		},
	)

	Field(13, "compression", String, "compression type",
		func() {
			Enum("gzip", "none", "default")
			Example("gzip")
			Example("none")
		},
	)

	Field(14, "num_flush_rows", Int, "number of rows for the export before data is flushed to the disk (persisted)",
		func() {
			Example(200000)
		},
	)

	Field(15, "num_flush_bytes", Int, "number of bytes for the export before data is flushed to the disk",
		func() {
			Example(2000000)
		},
	)

	Field(16, "num_concurrent_tables", Int, "number of tables to process at the same time; this is usually sized based on number of CPUs",
		func() {
			Example(4)
		},
	)

	Field(17, "num_batch_rows_export", Int, "number of rows to export at a given time for each iteration; tune this so that you get most out of CPU and can batch the most data together",
		func() {
			Example(100000)
		},
	)

	Field(18, "pg_logical_slot_name", String, "name for pg replication slot",
		func() {
			Example("")
		},
	)

	Field(19, "pg_logical_plugin", String, "name for pg replication plugin",
		func() {
			Example("")
		},
	)

	Field(20, "pg_drop_slot", Boolean, "if set and exists, drops the existing replication slot",
		func() {
			Example(false)
		},
	)

	Field(21, "name", String, "the name of the fetch run", func() {
		Example("rluu pg to cockroach")
	})

	Required(
		"source_conn",
		"target_conn",
		"mode",
		"store",
		"cleanup_intermediary_store",
		"local_path",
		"local_path_listen_address",
		"local_path_crdb_address",
		"bucket_name",
		"bucket_path",
		"log_file",
		"truncate",
		"compression",
		"num_flush_rows",
		"num_flush_bytes",
		"num_concurrent_tables",
		"num_batch_rows_export",
		"pg_logical_slot_name",
		"pg_logical_plugin",
		"pg_drop_slot",
		"name",
	)
})

var FetchRun = Type("fetch_run", func() {
	Field(1, "id", Int, "ID of the run",
		func() {
			Example(1704233521)
		},
	)

	Field(2, "name", String, "name of the fetch run", func() {
		Example("jyang pg to crdb")
	})

	Field(3, "status", String, "status of the fetch run", func() {
		Enum("IN_PROGRESS", "SUCCESS", "FAILURE")
		Example("IN_PROGRESS")
	})

	Field(4, "started_at", Int, "started at time",
		func() {
			Example(1704233519)
		},
	)

	Field(5, "finished_at", Int, "finished at time",
		func() {
			Example(1704233521)
		},
	)

	Required(
		"id",
		"name",
		"status",
		"started_at",
		"finished_at",
	)
})

var DetailedFetchStats = Type("fetch_stats_detailed", func() {
	Field(1, "percent_complete", String, "percentage complete of fetch run", func() {
		Example("jyang pg to crdb")
	})
	Field(2, "num_errors", Int, "number of errors processed",
		func() {
			Example(0)
		},
	)
	Field(3, "num_tables", Int, "number of tables processed",
		func() {
			Example(5)
		},
	)
	Field(4, "num_rows", Int, "number of rows",
		func() {
			Example(100000)
		},
	)
	Field(5, "net_duration_ms", Float64, "net duration in milliseconds",
		func() {
			Example(100000.00)
		},
	)
	Field(5, "import_duration_ms", Float64, "import duration in milliseconds",
		func() {
			Example(100000.00)
		},
	)
	Field(6, "export_duration_ms", Float64, "export duration in milliseconds",
		func() {
			Example(100000.00)
		},
	)
	Field(7, "cdc_cursor", String, "CDC cursor",
		func() {
			Example("0/3F3E0B8")
		},
	)

	Required(
		"percent_complete",
		"num_errors",
		"num_tables",
		"num_rows",
		"net_duration_ms",
		"import_duration_ms",
		"export_duration_ms",
		"cdc_cursor",
	)
})

var Log = Type("log", func() {
	Field(1, "timestamp", Int, "timestamp of log",
		func() {
			Example(1704233519)
		},
	)
	Field(2, "level", String, "level for logging", func() {
		Example("INFO")
	})
	Field(3, "message", String, "message for the logging", func() {
		Example("This is a log message")
	})

	Required(
		"timestamp",
		"level",
		"message",
	)
})

var FetchRunDetailed = Type("fetch_run_detailed", func() {
	Field(1, "id", Int, "ID of the run",
		func() {
			Example(1704233521)
		},
	)
	Field(2, "name", String, "name of the fetch run", func() {
		Example("jyang pg to crdb")
	})

	Field(3, "status", String, "status of the fetch run", func() {
		Enum("IN_PROGRESS", "SUCCESS", "FAILURE")
		Example("IN_PROGRESS")
	})

	Field(4, "started_at", Int, "started at time",
		func() {
			Example(1704233519)
		},
	)

	Field(5, "finished_at", Int, "finished at time",
		func() {
			Example(1704233521)
		},
	)

	Field(6, "stats", DetailedFetchStats, "fetch statistics")

	Field(7, "logs", ArrayOf(Log), "logs for fetch run")

	Field(8, "verify_runs", ArrayOf(VerifyRun), "verify runs linked to fetch runs")

	Required(
		"id",
		"name",
		"status",
		"started_at",
		"finished_at",
		"logs",
		"verify_runs",
	)
})

var VerifyRun = Type("verify_run", func() {
	Field(1, "id", Int, "ID of the run",
		func() {
			Example(1704233521)
		},
	)

	Field(2, "name", String, "name of the run", func() {
		Example("jyang pg to crdb")
	})

	Field(3, "status", String, "status of the run", func() {
		Enum("IN_PROGRESS", "SUCCESS", "FAILURE")
		Example("IN_PROGRESS")
	})

	Field(4, "started_at", Int, "started at time",
		func() {
			Example(1704233519)
		},
	)

	Field(5, "finished_at", Int, "finished at time",
		func() {
			Example(1704233521)
		},
	)

	Field(6, "fetch_id", Int, "ID of the associated fetch run",
		func() {
			Example(1704233521)
		},
	)

	Required(
		"id",
		"name",
		"status",
		"started_at",
		"finished_at",
		"fetch_id",
	)
})

var VerifyStats = Type("verify_stats_detailed", func() {
	Field(1, "num_tables", Int, "number of tables processed",
		func() {
			Example(5)
		},
	)
	Field(2, "num_truth_rows", Int, "number of rows processed",
		func() {
			Example(100000)
		},
	)
	Field(3, "num_success", Int, "number of successful rows processed",
		func() {
			Example(50000)
		},
	)
	Field(4, "num_conditional_success", Int, "number of rows that had conditional success",
		func() {
			Example(100000)
		},
	)
	Field(5, "num_missing", Int, "number of missing rows",
		func() {
			Example(1)
		},
	)
	Field(6, "num_mismatch", Int, "number of mismatching rows",
		func() {
			Example(1)
		},
	)
	Field(7, "num_extraneous", Int, "number of extraneous rows",
		func() {
			Example(1)
		},
	)
	Field(8, "num_live_retry", Int, "number of live retries",
		func() {
			Example(1)
		},
	)
	Field(9, "num_column_mismatch", Int, "number column mismatches",
		func() {
			Example(1)
		},
	)
	Field(10, "net_duration_ms", Float64, "net duration in milliseconds",
		func() {
			Example(100000.00)
		},
	)

	Required(
		"num_tables",
		"num_truth_rows",
		"num_success",
		"num_conditional_success",
		"num_missing",
		"num_mismatch",
		"num_extraneous",
		"num_live_retry",
		"num_column_mismatch",
		"net_duration_ms",
	)
})

var VerifyMismatch = Type("verify_mismatch", func() {
	Field(1, "timestamp", Int, "timestamp of log",
		func() {
			Example(1704233519)
		},
	)
	Field(2, "level", String, "level for logging", func() {
		Example("INFO")
	})
	Field(3, "message", String, "message for the logging", func() {
		Example("This is a log message")
	})
	Field(4, "schema", String, "schema for the db", func() {
		Example("public")
	})
	Field(5, "table", String, "name of the table", func() {
		Example("users")
	})
	Field(6, "type", String, "type of mismatch", func() {
		Example("mismatching table definition")
	})

	Required(
		"timestamp",
		"level",
		"message",
		"schema",
		"table",
		"type",
	)
})

var VerifyRunDetailed = Type("verify_run_detailed", func() {
	Field(1, "id", Int, "ID of the run",
		func() {
			Example(1704233521)
		},
	)

	Field(2, "name", String, "name of the run", func() {
		Example("jyang pg to crdb")
	})

	Field(3, "status", String, "status of the run", func() {
		Enum("IN_PROGRESS", "SUCCESS", "FAILURE")
		Example("IN_PROGRESS")
	})

	Field(4, "started_at", Int, "started at time",
		func() {
			Example(1704233519)
		},
	)

	Field(5, "finished_at", Int, "finished at time",
		func() {
			Example(1704233521)
		},
	)

	Field(6, "fetch_id", Int, "ID of the associated fetch run",
		func() {
			Example(1704233521)
		},
	)

	Field(7, "stats", VerifyStats, "verify statistics")

	Field(8, "mismatches", ArrayOf(VerifyMismatch), "verify mismatches (i.e. data mismatches, missing rows)")

	Required(
		"id",
		"name",
		"status",
		"started_at",
		"finished_at",
		"fetch_id",
		"stats",
		"mismatches",
	)
})

var FetchAttemptID = Type("fetch_attempt_id", Int, func() {
	Description("the id of a fetch attempt")
})

var VerifyAttemptID = Type("verify_attempt_id", Int, func() {
	Description("the id of a verify attempt")
})
