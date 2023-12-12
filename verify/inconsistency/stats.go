package inconsistency

import (
	"fmt"

	"github.com/rs/zerolog"
)

// RowStats includes all details about the total rows processed.
type RowStats struct {
	Schema        string
	Table         string
	NumVerified   int
	NumSuccess    int
	NumMissing    int
	NumMismatch   int
	NumExtraneous int
	NumLiveRetry  int
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
	l.Info().
		Str("table_schema", s.Schema).
		Str("table_name", s.Table).
		Int("num_truth_rows", s.NumVerified).
		Int("num_success", s.NumSuccess).
		Int("num_missing", s.NumMissing).
		Int("num_mismatch", s.NumMismatch).
		Int("num_extraneous", s.NumExtraneous).
		Int("num_live_retry", s.NumLiveRetry).
		Msg(m)
}
