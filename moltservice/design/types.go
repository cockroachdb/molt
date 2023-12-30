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
	// TODO: make this an enum
	Field(1, "mode", String, "Mode of operation for fetch",
		func() {
			Example("IMPORT INTO")
			Example("COPY FROM")
			Example("DIRECT COPY")
		},
	)

	// TODO: make this an enum
	Field(2, "store", String, "Type of intermediary store",
		func() {
			Example("AWS")
			Example("GCP")
			Example("Local")
		},
	)

	Field(3, "cleanup_intermediary_store", Boolean, "whether the intermediate store should be cleaned up after the fetch task",
		func() {
			Example(true)
		},
	)

	Field(4, "local_path", String, "the absolute or relative path to write export files",
		func() {
			Example("/usr/Documents/fetch")
		},
	)

	Field(5, "local_path_listen_address", String, "the local address where the file server will be spun up",
		func() {
			Example("http://localhost:5000")
		},
	)

	Field(6, "local_path_crdb_address", String, "the local address CRDB will use to access the import files",
		func() {
			Example("http://localhost:5000")
		},
	)

	Field(7, "bucket_name", String, "the local address CRDB will use to access the import files",
		func() {
			Example("http://localhost:5000")
		},
	)

	Field(8, "bucket_path", String, "the sub-path within the bucket to write the export files",
		func() {
			Example("fetch/export")
		},
	)

	Field(9, "log_file", String, "if specified, writes task execution logs to a file and stdout; otherwise, just writes to stdout",
		func() {
			Example("task.log")
		},
	)

	Field(10, "truncate", Boolean, "if specified, truncates the target tables before running the data load",
		func() {
			Example(true)
		},
	)
})

var FetchAttemptID = Type("fetch_attempt_id", Int, func() {
	Description("the id of a fetch attempt")
})
