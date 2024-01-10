// Copyright 2024 Cockroach Labs Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

// flagEnabled prevents tests from running from `go test ./...` by default.
// We avoid build tags here as it would require adding it to every test.
var flagEnabled = flag.Bool("e2e-enabled", false, "whether the test should execute")

func TestMain(t *testing.M) {
	flag.Parse()

	// If tests are not enabled, abort.
	if !*flagEnabled {
		if _, err := fmt.Fprintln(os.Stderr, "e2e tests are not enabled (use -e2e-enabled)"); err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	os.Exit(t.Run())
}
