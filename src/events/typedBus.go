package events

import (
	devcodeerror "DevCode/src/DevCodeError"
	"DevCode/src/constants"
	"sync"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

func NewTypedBus[T any](pool *ants.Pool, logger *zap.Logger) *TypedBus[T] {
	return &TypedBus[T]{
		pool:     pool,
		handlers: make(map[constants.Source]func(Event[T])),
		logger:   logger,
	}
}

type TypedBus[T any] struct {
	handlers     map[constants.Source]func(Event[T])
	pool         *ants.Pool
	handlerMutex sync.RWMutex
	logger       *zap.Logger
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
						instance.logger.Error("", zap.Error(devcodeerror.Wrap(nil, devcodeerror.FailHandleEvent, "Panic with handle event")), zap.Any("recover", recover))
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
