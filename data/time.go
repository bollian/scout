package data

import (
	"fmt"
	"time"
)

// Timestamp represents a time in UTC
type Timestamp struct {
	time time.Time
}

// Now gets the current Timestamp
func Now() Timestamp {
	return Timestamp{time: time.Now().UTC()}
}

// String returns the string representation of the Timestamp
func (ts Timestamp) String() string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		ts.time.Year(), ts.time.Month(), ts.time.Day(),
		ts.time.Hour(), ts.time.Minute(), ts.time.Second())
}

// Local converts the Timestamp into the standard Go representation of time in
// the local timezone.
func (ts Timestamp) Local() time.Time {
	return ts.time.Local()
}

// Add modifies the Timestamp by the given duration
func (ts Timestamp) Add(dur time.Duration) Timestamp {
	return Timestamp{time: ts.time.Add(dur)}
}

// Year retrieves the year of the Timestamp
func (ts Timestamp) Year() int {
	return ts.time.Year()
}
