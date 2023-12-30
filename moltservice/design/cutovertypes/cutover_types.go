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

package cutovertypes

//go:generate go run golang.org/x/tools/cmd/stringer -type=CutoverType
type CutoverType int

const (
	Consistent CutoverType = iota + 1
	Immediate
	cutoverTypeLimit
)

func CutoverTypeArray() []any {
	res := make([]any, 0)
	for t := CutoverType(1); t < cutoverTypeLimit; t++ {
		res = append(res, t.String())
	}
	return res
}
