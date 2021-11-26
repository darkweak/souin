package rfc

import (
	"math"
	"time"
)

func correctedInitialAge(responseTime, dateValue time.Time) int {
	apparentAge := responseTime.Sub(dateValue)
	if apparentAge < 0 {
		apparentAge = 0
	}

	return int(math.Ceil(apparentAge.Seconds()))
}
