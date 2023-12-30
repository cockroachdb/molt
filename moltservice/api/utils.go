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

package api

import (
	"fmt"
	"regexp"

	"github.com/cockroachdb/molt/moltservice/gen/http/moltservice/server"
	"github.com/coreos/go-semver/semver"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const failedToConnectErrorPattern = "cannot connect to the LMS from the orchestrator for dialect %s"
const failedToParseUrlErrorPattern = "cannot parse the url to LMS from the orchestrator for dialect %s"

const parseErrorPattern = "cannot parse the connection string for %s"

// PathPatternDetails holds information about the original path pattern
// and the regexp pattern to match.
type PathPatternDetails struct {
	original      string
	regExpPattern string
}

var wildSeg = regexp.MustCompile(`/{([a-zA-Z0-9_]+)}`)

// replaceWithPattern replaces the path pattern interpolation string with
// the associated regexp wildcard so that we can easily do string matches
// on incoming paths.
func replacePathWithPattern(path string) string {
	return wildSeg.ReplaceAllString(path, "/[a-zA-Z0-9-_]+")
}

// getPathPatternDetail gets the path pattern details for a given mount point
// by keying off the mount point pattern and replacing the pattern
// with the relevant regular expression.
func getPathPatternDetails(mounts []*server.MountPoint) []*PathPatternDetails {
	patternDtls := make([]*PathPatternDetails, len(mounts))

	for i, dtl := range mounts {
		patternDtls[i] = &PathPatternDetails{
			original:      dtl.Pattern,
			regExpPattern: replacePathWithPattern(dtl.Pattern),
		}
	}

	return patternDtls
}

// findMatchingPattern finds the matching pattern string from the endpoint details and returns
// the original matched pattern. If one cannot be found, it returns an empty string.
func findMatchingPattern(path string, dtls []*PathPatternDetails) (string, error) {
	for _, dtl := range dtls {
		// Make sure that things strictly start and end with this string.
		cleanedRegexPath := fmt.Sprintf("^%s$", dtl.regExpPattern)
		regex, err := regexp.Compile(cleanedRegexPath)
		if err != nil {
			return "", err
		}

		if regex.MatchString(path) {
			return dtl.original, nil
		}
	}

	return "", nil
}

// checkVersionsMatch checks to see if the semvers match. If not, returns an actionable help string.
func checkVersionsMatch(clientVer, serverVer string) string {
	client := semver.New(clientVer)
	server := semver.New(serverVer)

	if client.LessThan(*server) {
		return fmt.Sprintf("Please upgrade your client version to %s.", serverVer)
	} else if server.LessThan(*client) {
		return fmt.Sprintf("Please upgrade your server version to %s.", clientVer)
	}

	return ""
}
