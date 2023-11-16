// Copyright 2022 Cockroach Labs Inc.
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

package main

import (
	"bufio"
	"flag"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path"
	"strings"
)

type version struct {
	Name string
	Link string
}

type renderData struct {
	Title    string
	Versions []*version
}

const (
	inputName  = "versions.txt"
	tmplName   = "climanifestHtml.tmpl"
	outputName = "versions.html"
	titleName  = ""
)

var versionFile string
var outputFile string
var templateFile string
var title string

func init() {
	flag.StringVar(&versionFile, "version-file", inputName, "the input versions file")
	flag.StringVar(&outputFile, "output-file", outputName, "the output version file")
	flag.StringVar(&templateFile, "template-file", tmplName, "the template file")
	flag.StringVar(&title, "title", titleName, "the title for the manifest")
	flag.Parse()
}

func readFromVersions(versionsFile string) ([]*version, error) {
	file, err := os.Open(versionsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	seen := map[string]bool{}
	versions := []*version{}

	// Loop through the lines
	for scanner.Scan() {
		urlStr := strings.TrimSpace(scanner.Text())

		if urlStr == "" || !strings.Contains(urlStr, ".tgz") {
			continue
		}
		// Do something with the line, e.g., print it

		u, err := url.Parse(urlStr)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			return nil, err
		}

		// If we've already seen the path, we should skip.
		if _, ok := seen[u.Path]; ok {
			fmt.Printf("Skipping, already saw path: %s\n", u.Path)
			continue
		}

		seen[u.Path] = true
		versions = append(versions, &version{
			Name: path.Base(u.Path),
			Link: urlStr,
		})
	}

	return versions, nil
}

func main() {
	versions, err := readFromVersions(versionFile)
	if err != nil {
		panic(err)
	}

	funcMap := template.FuncMap{
		"dec":     func(i int) int { return i - 1 },
		"replace": strings.ReplaceAll,
	}
	var tmplFile = templateFile
	tmpl, err := template.New(tmplName).Funcs(funcMap).ParseFiles(tmplFile)
	if err != nil {
		panic(err)
	}
	var f *os.File
	f, err = os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(f, renderData{
		Title:    title,
		Versions: versions,
	})
	if err != nil {
		panic(err)
	}
	err = f.Close()
	if err != nil {
		panic(err)
	}
}
