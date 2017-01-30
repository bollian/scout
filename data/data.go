package data

import (
	"time"
)

// VerifyYear checks to see that the provided game year is supported by
// the scouting system.
func VerifyYear(year int) bool {
	return year >= MinYear && year <= time.Now().Year()
}
