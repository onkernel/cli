package util

import "time"

// DefaultTimeLayout is the standard layout used for displaying timestamps.
// Includes the local timezone abbreviation to make it clear times are local.
const DefaultTimeLayout = "2006-01-02 15:04:05 MST"

// FormatLocal formats the provided time in the user's local timezone.
// If the time is zero, it returns "-".
func FormatLocal(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.In(time.Local).Format(DefaultTimeLayout)
}
