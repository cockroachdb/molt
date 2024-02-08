package utils

import (
	"testing"
	"time"

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

func TestFormatMillisecondsToTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{
			name:     "seconds duration",
			input:    20 * time.Second,
			expected: "000h 00m 20s",
		},
		{
			name:     "minutes + seconds duration",
			input:    80 * time.Second,
			expected: "000h 01m 20s",
		},
		{
			name:     "minutes duration",
			input:    20 * time.Minute,
			expected: "000h 20m 00s",
		},
		{
			name:     "hours + minutes duration",
			input:    80 * time.Minute,
			expected: "001h 20m 00s",
		},
		{
			name:     "hours duration",
			input:    20 * time.Hour,
			expected: "020h 00m 00s",
		},
		{
			name:     "hours + minutes + seconds duration",
			input:    80*time.Minute + 10*time.Second,
			expected: "001h 20m 10s",
		},
		{
			name:     "weeks duration",
			input:    24 * 7 * 2 * time.Hour,
			expected: "336h 00m 00s",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val := FormatDurationToTimeString(tc.input)
			require.Equal(t, tc.expected, val)
		})
	}
}
