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

import (
	"fmt"

	. "goa.design/goa/v3/dsl"
	cors "goa.design/plugins/v3/cors/dsl"
)

const (
	BaseAPIPath = "/api/v1"
)

var (
	FetchBaseAPIPath  = fmt.Sprintf("%s/fetch", BaseAPIPath)
	VerifyBaseAPIPath = fmt.Sprintf("%s/verify", BaseAPIPath)
)

var _ = API("moltservice", func() {
	Title("MOLT Service")
	Description("Service for coordinating between clients (CLI + Web UI) and MOLT tooling")
	Server("moltservice", func() {
		Host("localhost", func() {
			URI("http://localhost:4500")
		})
	})
})

var _ = Service("moltservice", func() {
	Description("MOLT service performs operations using MOLT tooling")

	HTTP(func() {})

	// TODO (rluu) #462: need to make this more secure (pass in this as secret later)
	// The "$" prefix tells the CORS handler to look for the allowed host pattern
	// in the env variable.
	cors.Origin("$MOLT_SERVICE_ALLOW_ORIGIN", func() {
		cors.Methods("GET", "POST")
		cors.Headers("Content-Type")
		cors.Credentials()
	})

	Method("create_fetch_task", func() {
		Payload(CreateFetchPayload)

		HTTP(func() {
			POST(FetchBaseAPIPath)
		})

		Result(FetchAttemptID)
	})

	Method("get_fetch_tasks", func() {
		HTTP(func() {
			GET(FetchBaseAPIPath)
		})

		Result(ArrayOf(FetchRun))
	})

	Method("get_specific_fetch_task", func() {
		Payload(func() {
			Field(1, "id", Int, "id for the fetch task")
			Required("id")
		})

		HTTP(func() {
			GET(fmt.Sprintf("%s/{id}", FetchBaseAPIPath))
		})

		Result(FetchRunDetailed)
	})

	Method("create_verify_task_from_fetch", func() {
		Payload(func() {
			Field(1, "id", Int, "id for the fetch task")
			Required("id")
		})

		HTTP(func() {
			POST(fmt.Sprintf("%s/{id}/verify", FetchBaseAPIPath))
		})

		Result(VerifyAttemptID)
	})

	// OpenAPI spec.
	Files("/openapi.json", "./gen/http/openapi.json")
	// RapiDoc UI.
	Files("/docs.html", "./assets/docs.html")
})
