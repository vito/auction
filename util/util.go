package util

import (
	"fmt"
	"math/rand"
	"time"
)

var R *rand.Rand
var guidTracker map[string]int

func init() {
	R = rand.New(rand.NewSource(time.Now().UnixNano()))
	ResetGuids()
}

func ResetGuids() {
	guidTracker = map[string]int{}
}

func NewGuid(prefix string) string {
	guidTracker[prefix] = guidTracker[prefix] + 1
	return fmt.Sprintf("%s-%d", prefix, guidTracker[prefix])
}

func RandomSleep(min time.Duration, max time.Duration, timeout time.Duration) bool {
	sleepDuration := time.Duration(R.Float64()*float64(max-min) + float64(min))
	if sleepDuration <= timeout {
		time.Sleep(sleepDuration)
		return true
	} else {
		time.Sleep(timeout)
		return false
	}
}
