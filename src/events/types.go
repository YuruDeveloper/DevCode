package events

import (
	"DevCode/src/constants"
	"time"
)

type Event[T any] struct {
	Data      T
	TimeStamp time.Time
	Source    constants.Source
}
