package events

import (
	"DevCode/constants"
	"sync"
	"time"
)

type Event[T any] struct {
	Data      T
	TimeStamp time.Time
	Source    constants.Source
}

func NewTypedBus[T any]() *TypedBus[T] {
	return &TypedBus[T]{
		handlers: make(map[constants.Source]func(Event[T])),
	}
}

type TypedBus[T any] struct {
	handlers     map[constants.Source]func(Event[T])
	handlerMutex sync.RWMutex
}
