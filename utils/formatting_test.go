package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchesFileConvention(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     bool
	}{
		{
			name:     "matching csv file",
			fileName: "part_00000004.csv",
			want:     true,
		},
		{
			name:     "matching gzip file",
			fileName: "part_00000004.tar.gz",
			want:     true,
		},
		{
			name:     "non-matching csv file because wrong number of digits",
			fileName: "part_0000004.tar.gz",
			want:     false,
		},
		{
			name:     "non-matching file but similar",
			fileName: "part_djdkfjkd.tar.gz",
			want:     false,
		},
		{
			name:     "non-matching file completely different",
			fileName: "pakl;sdjf;alksdjf;alksdf",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := MatchesFileConvention(tt.fileName)
			require.Equal(t, tt.want, actual)
		})
	}
}
