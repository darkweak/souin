package rfc

import (
	"math"
	"strconv"
	"time"
)

func correctedInitialAge(responseTime, dateValue time.Time) time.Duration {
	apparentAge := responseTime.Sub(dateValue)
	if apparentAge < 0 {
		apparentAge = 0
	}

	return apparentAge
}

func ageToString(age time.Duration) string {
	return strconv.Itoa(int(math.Ceil(age.Seconds())))
}
