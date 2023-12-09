package utils

import (
	"fmt"
	"time"
)

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
