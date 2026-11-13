package timer

import "time"

func get10Ms() time.Duration {
	return time.Duration(int64(time.Now().UnixNano() / int64(time.Millisecond) / 10))
}
