//go:build tools
// +build tools

package main

import (
	_ "github.com/alvaroloes/enumer"
	_ "github.com/cockroachdb/crlfmt"
	_ "github.com/jstemmer/go-junit-report"
	_ "goa.design/goa/v3/cmd/goa"
)
