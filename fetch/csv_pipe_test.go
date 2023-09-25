package fetch

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cockroachdb/molt/dbtable"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestCSVPipe(t *testing.T) {
	for _, tc := range []struct {
		desc      string
		toWrite   string
		files     []string
		flushSize int
		flushRows int
	}{
		{
			desc: "one big file",
			toWrite: `1,abcd,efgh
2,efgh,""""
3,%,g
`,
			files: []string{
				`1,abcd,efgh
2,efgh,""""
3,%,g
`,
			},
			flushSize: 1024,
		},
		{
			desc: "split files",
			toWrite: `1,a
2,bbbb
3,cc
4,a
`,
			files: []string{
				`1,a
2,bbbb
`,
				`3,cc
`,
				`4,a
`,
			},
			flushSize: 4,
		},
		{
			desc: "quoted new lines",
			toWrite: `1,a,"this is
a
multiline part"
2,a,c`,
			files: []string{
				`1,a,"this is
a
multiline part"
`,
				`2,a,c
`,
			},
			flushSize: 4,
		},
		{
			desc: "flush after 1 row",
			toWrite: `1,abcd,efgh
2,efgh,""""
3,%,g`,
			files: []string{
				"1,abcd,efgh\n",
				`2,efgh,""""
`,
				`3,%,g
`,
			},
			flushSize: 1024,
			flushRows: 1,
		},
		{
			desc: "flush after two rows",
			toWrite: `1,abcd,efgh
2,efgh,""""
3,%,g`,
			files: []string{
				"1,abcd,efgh\n2,efgh,\"\"\"\"\n",
				"3,%,g\n",
			},
			flushSize: 1024,
			flushRows: 2,
		},
		{
			desc: "flush after multiple rows",
			toWrite: `1,abcd,efgh
2,efgh,""""
3,%,g`,
			files: []string{
				`1,abcd,efgh
2,efgh,""""
3,%,g
`,
			},
			flushSize: 1024,
			flushRows: 4,
		},
		{
			desc: "flush after mix of flush size and flush rows",
			toWrite: `1,abcd,efgh
2,efgh,""""
3,%,g
4,a,b
`,
			files: []string{
				`1,abcd,efgh
`,
				`2,efgh,""""
3,%,g
`,
				`4,a,b
`,
			},
			flushSize: 10,
			flushRows: 2,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			var bufs []testStringBuf
			pipe := newCSVPipe(
				strings.NewReader(tc.toWrite),
				zerolog.New(os.Stdout),
				tc.flushSize,
				tc.flushRows,
				func() io.WriteCloser {
					bufs = append(bufs, testStringBuf{})
					return &bufs[len(bufs)-1]
				},
			)
			require.NoError(t, pipe.Pipe(dbtable.Name{Schema: "test", Table: "test"}))
			var written []string
			for _, buf := range bufs {
				written = append(written, buf.String())
			}
			require.Equal(t, tc.files, written)
		})
	}
}

type testStringBuf struct {
	strings.Builder
}

func (b *testStringBuf) Close() error {
	return nil
}
