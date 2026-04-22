package helper

import "time"

func DefaultLocation() *time.Location {
	// Load location
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("Asia/Bangkok", (7 * 3600))
	}

	return loc
}
