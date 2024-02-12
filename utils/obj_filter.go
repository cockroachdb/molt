package utils

import (
	"regexp"

	"github.com/cockroachdb/molt/dbtable"
)

const DefaultFilterString = ".*"

type FilterString = string

func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		SchemaFilter: DefaultFilterString,
		TableFilter:  DefaultFilterString,
	}
}

type FilterConfig struct {
	SchemaFilter FilterString
	TableFilter  FilterString
}

func FilterResult(cfg FilterConfig, r Result) (Result, error) {
	if cfg.SchemaFilter == DefaultFilterString && cfg.TableFilter == DefaultFilterString {
		return r, nil
	}
	schemaRe, err := regexp.CompilePOSIX(cfg.SchemaFilter)
	if err != nil {
		return r, err
	}
	tableRe, err := regexp.CompilePOSIX(cfg.TableFilter)
	if err != nil {
		return r, err
	}
	newResult := Result{
		Verified:         r.Verified[:0],
		MissingTables:    r.MissingTables[:0],
		ExtraneousTables: r.ExtraneousTables[:0],
	}
	for _, v := range r.Verified {
		if MatchesFilter(v[0].Name, schemaRe, tableRe) {
			newResult.Verified = append(newResult.Verified, v)
		}
	}
	for _, t := range r.MissingTables {
		if MatchesFilter(t.Name, schemaRe, tableRe) {
			newResult.MissingTables = append(newResult.MissingTables, t)
		}
	}
	for _, t := range r.ExtraneousTables {
		if MatchesFilter(t.Name, schemaRe, tableRe) {
			newResult.ExtraneousTables = append(newResult.ExtraneousTables, t)
		}
	}
	return newResult, nil
}

func MatchesFilter(n dbtable.Name, schemaRe, tableRe *regexp.Regexp) bool {
	return schemaRe.MatchString(string(n.Schema)) && tableRe.MatchString(string(n.Table))
}

type Result struct {
	Verified [][2]dbtable.DBTable

	MissingTables    []MissingTable
	ExtraneousTables []ExtraneousTable
}

// MissingTable represents a table that is missing from a database.
type MissingTable struct {
	dbtable.DBTable
}

// ExtraneousTable represents a table that is extraneous to a database.
type ExtraneousTable struct {
	dbtable.DBTable
}

func (r *Result) AllTableFromSource() []dbtable.DBTable {
	var res []dbtable.DBTable

	for _, vTs := range r.Verified {
		res = append(res, vTs[0])
	}

	for _, mT := range r.MissingTables {
		res = append(res, mT.DBTable)
	}

	return res
}
