package inconsistency

import (
	"fmt"

	"github.com/rs/zerolog"
)

// RowStats includes all details about the total rows processed.
type RowStats struct {
	Schema                string
	Table                 string
	NumVerified           int
	NumSuccess            int
	NumConditionalSuccess int
	NumMissing            int
	NumMismatch           int
	NumColumnMismatch     int
	NumExtraneous         int
	NumLiveRetry          int
}

func (s *RowStats) String() string {
	return fmt.Sprintf(
		"truth rows seen: %d, success: %d, missing: %d, mismatch: %d, extraneous: %d, live_retry: %d",
		s.NumVerified,
		s.NumSuccess,
		s.NumMissing,
		s.NumMismatch,
		s.NumExtraneous,
		s.NumLiveRetry,
	)
}

// reportRunningSummary reports the number of total rows and errors seen
// during the execution of verify.
func reportRunningSummary(l zerolog.Logger, s RowStats, m string) {
	if s.NumConditionalSuccess > 0 {
		m = fmt.Sprintf("%s - please check all warnings and errors for column mismatches to determine success", m)
	}

	l.Info().
		Str("table_schema", s.Schema).
		Str("table_name", s.Table).
		Int("num_truth_rows", s.NumVerified).
		Int("num_success", s.NumSuccess).
		Int("num_conditional_success", s.NumConditionalSuccess).
		Int("num_missing", s.NumMissing).
		Int("num_mismatch", s.NumMismatch).
		Int("num_extraneous", s.NumExtraneous).
		Int("num_live_retry", s.NumLiveRetry).
		Int("num_column_mismatch", s.NumColumnMismatch).
		Msg(m)
}
