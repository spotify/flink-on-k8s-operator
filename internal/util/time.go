package util

import (
	"fmt"
	"time"
)

// TimeConverter converts between time.Time and string.
type TimeConverter struct{}

// FromString converts string to time.Time.
func (tc *TimeConverter) FromString(timeStr string) time.Time {
	timestamp, err := time.Parse(
		time.RFC3339, timeStr)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse time string: %s", timeStr))
	}
	return timestamp
}

// ToString converts time.Time to string.
func (tc *TimeConverter) ToString(timestamp time.Time) string {
	return timestamp.Format(time.RFC3339)
}

// SetTimestamp sets the current timestamp to the target.
func SetTimestamp(target *string) {
	var tc = &TimeConverter{}
	var now = time.Now()
	*target = tc.ToString(now)
}

func GetTime(timeStr string) time.Time {
	var tc TimeConverter
	return tc.FromString(timeStr)
}
