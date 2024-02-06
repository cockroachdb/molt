package utils

import (
	"testing"

	"github.com/cockroachdb/molt/dbtable"
	"github.com/stretchr/testify/require"
)

func TestFilterResult(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		config   FilterConfig
		r        Result
		expected Result
	}{
		{
			desc:   "default config doesn't filter anything",
			config: DefaultFilterConfig(),
			r: Result{
				Verified: [][2]dbtable.DBTable{
					{
						{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}},
						{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}},
					},
					{
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
					},
					{
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
					},
				},
				MissingTables: []MissingTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
				ExtraneousTables: []ExtraneousTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
			},
			expected: Result{
				Verified: [][2]dbtable.DBTable{
					{
						{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}},
						{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}},
					},
					{
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
					},
					{
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
					},
				},
				MissingTables: []MissingTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
				ExtraneousTables: []ExtraneousTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
			},
		},
		{
			desc: "table filter",
			config: FilterConfig{
				TableFilter:  "b",
				SchemaFilter: DefaultFilterString,
			},
			r: Result{
				Verified: [][2]dbtable.DBTable{
					{
						{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}},
						{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}},
					},
					{
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
					},
					{
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
					},
				},
				MissingTables: []MissingTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
				ExtraneousTables: []ExtraneousTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
			},
			expected: Result{
				Verified: [][2]dbtable.DBTable{
					{
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
					},
					{
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
					},
				},
				MissingTables: []MissingTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
				},
				ExtraneousTables: []ExtraneousTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
				},
			},
		},
		{
			desc: "schema filter",
			config: FilterConfig{
				SchemaFilter: "c",
				TableFilter:  DefaultFilterString,
			},
			r: Result{
				Verified: [][2]dbtable.DBTable{
					{
						{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}},
						{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}},
					},
					{
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}},
					},
					{
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
					},
				},
				MissingTables: []MissingTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
				ExtraneousTables: []ExtraneousTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "aaa"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "aaa", Table: "bbb"}}},
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
			},
			expected: Result{
				Verified: [][2]dbtable.DBTable{
					{
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
						{Name: dbtable.Name{Schema: "ccc", Table: "bbb"}},
					},
				},
				MissingTables: []MissingTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
				ExtraneousTables: []ExtraneousTable{
					{DBTable: dbtable.DBTable{Name: dbtable.Name{Schema: "ccc", Table: "aaa"}}},
				},
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			r, err := FilterResult(tc.config, tc.r)
			require.NoError(t, err)
			require.Equal(t, tc.expected, r)
		})
	}
}
