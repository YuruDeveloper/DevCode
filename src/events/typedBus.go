package events

import (
	"DevCode/src/constants"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/panjf2000/ants/v2"
)

func NewTypedBus[T any](pool *ants.Pool) *TypedBus[T] {
	return &TypedBus[T]{
		pool:     pool,
		handlers: make(map[constants.Source]func(Event[T])),
	}
}

type TypedBus[T any] struct {
	handlers     map[constants.Source]func(Event[T])
	pool         *ants.Pool
	handlerMutex sync.RWMutex
}

func (instance *TypedBus[T]) Subscribe(source constants.Source, handler func(Event[T])) {
	instance.handlerMutex.Lock()
	defer instance.handlerMutex.Unlock()
	instance.handlers[source] = handler
}

func (instance *TypedBus[T]) UnSubscribe(source constants.Source) {
	instance.handlerMutex.Lock()
	defer instance.handlerMutex.Unlock()
	delete(instance.handlers, source)
}

func (instance *TypedBus[T]) Publish(event Event[T]) {
	instance.handlerMutex.Lock()
	defer instance.handlerMutex.Unlock()
	for _, handler := range instance.handlers {
		instance.pool.Submit(
			func() {
				defer func() {
					if recover := recover(); recover != nil {
						fmt.Printf("PANIC %v\n %s\n", recover, debug.Stack())
					}
				}()
				handler(event)
			},
		)
	}
}

func (instance *TypedBus[T]) Close() {
	instance.handlerMutex.Lock()
	defer instance.handlerMutex.Unlock()
	clear(instance.handlers)
}
