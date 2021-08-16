package rfc

import (
	"math"
	"strconv"
	"time"
)

func correctedInitialAge(responseTime, dateValue, requestTime time.Time, ageValue string) time.Duration {
	apparentAge := responseTime.Sub(dateValue)
	if apparentAge < 0 {
		apparentAge = 0
	}

	var initialAge time.Duration
	if ageValue != "" {
		if iAgeValue, _ := strconv.Atoi(ageValue); iAgeValue != 0 {
			initialAge = time.Duration(iAgeValue) * time.Second
		}
	}

	responseDelay := responseTime.Sub(requestTime)
	correctedAgeValue := initialAge + responseDelay

	if apparentAge > correctedAgeValue {
		return apparentAge
	}
	return correctedAgeValue
}

func ageToString(age time.Duration) string {
	return strconv.Itoa(int(math.Ceil(age.Seconds())))
}
