package utils

import (
	"fmt"
	"regexp"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
)

func SchemaTableString(schema, table tree.Name) string {
	return fmt.Sprintf("%s.%s", string(schema), string(table))
}

var FileConventionRegex = regexp.MustCompile(`part_[\d+]{8}(\.csv|\.tar\.gz)`)

func MatchesFileConvention(fileName string) bool {
	return FileConventionRegex.MatchString(fileName)
}

func FormatDurationToTimeString(duration time.Duration) string {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	// Hours are rounded to 3 places because there could be table loads that
	// take weeks, which could be hundreds of hours.
	// We don't show milliseconds because it's such a minimal amount of time
	// and is unlikely for most production tables. Also, if folks want
	// milliseconds, we are still logging out the milliseconds data side by side.
	return fmt.Sprintf("%03dh %02dm %02ds", hours, minutes, seconds)
}

// MaybeFormatDurationForTest is to make a deterministic duration for test.
func MaybeFormatDurationForTest(testOnly bool, duration time.Duration) time.Duration {
	if !testOnly {
		return duration
	}
	return time.Second
}

// MaybeFormatCDCCursor is to make a deterministic CDC cursor for test.
func MaybeFormatCDCCursor(testOnly bool, s string) string {
	if !testOnly {
		return s
	}
	return "0/19E3610"
}

// MaybeFormatFetchID is to make a deterministic fetch id for test.
func MaybeFormatFetchID(testOnly bool, s string) string {
	if !testOnly {
		return s
	}

	return "0000000000"
}

// MaybeFormatID is to make a deterministic UUID for test.
func MaybeFormatID(testOnly bool, s uuid.UUID) uuid.UUID {
	if !testOnly {
		return s
	}

	return uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440000"))
}
