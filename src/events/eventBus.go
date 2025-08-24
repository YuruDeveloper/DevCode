package events

import (
	"UniCode/src/constants"
	"fmt"
	"sync"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

func NewEventBus(logger *zap.Logger) (*EventBus, error) {
	pool, err := ants.NewPool(10000,ants.WithPreAlloc(true))
	if err != nil {
		return nil, fmt.Errorf("fail to create ants pool : %w", err)
	}
	return &EventBus{
		logger:      logger,
		pool:        pool,
		subscribers: make(map[EventType][]Subscriber, 6),
		busMutex:    sync.RWMutex{},
	}, nil
}

type EventBus struct {
	logger      *zap.Logger
	pool        *ants.Pool
	subscribers map[EventType][]Subscriber
	busMutex    sync.RWMutex
}

func (instance *EventBus) Subscribe(eventType EventType, subscriber Subscriber) {
	instance.busMutex.Lock()
	defer instance.busMutex.Unlock()
	instance.logger.Info("Event subscriber",
		zap.String("Type", eventType.String()),
		zap.String("subscriber", subscriber.GetID().String()),
	)

	instance.subscribers[eventType] = append(instance.subscribers[eventType], subscriber)
}

func (instance *EventBus) UnSubscribe(eventType EventType, subscriberID constants.Source) {
	instance.busMutex.Lock()
	defer instance.busMutex.Unlock()

	for index, subscriber := range instance.subscribers[eventType] {
		if subscriber.GetID() == subscriberID {
			instance.subscribers[eventType] = append(instance.subscribers[eventType][:index], instance.subscribers[eventType][index+1:]...)
			return
		}
	}
}

func (instance *EventBus) Publish(event Event) {
	instance.busMutex.RLock()
	defer instance.busMutex.RUnlock()
	for _, subscriber := range instance.subscribers[event.Type] {
		instance.pool.Submit(
			func() {
				defer func() {
					if recover := recover(); recover != nil {
						instance.logger.Error("Panicin subcriber",
							zap.String("Source", subscriber.GetID().String()),
							zap.Any("recover", recover),
						)
					}
				}()
				subscriber.HandleEvent(event)
			},
		)
	}
}

func (instance *EventBus) Close() {
	instance.pool.Release()
}
