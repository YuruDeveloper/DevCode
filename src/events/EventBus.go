package events

import (
	"UniCode/src/types"
	"sync"
)

func NewEventBus() *EventBus {
	return &EventBus{
		Subscribers: make(map[EventType][]Subscriber),
	}
}

type EventBus struct {
	Subscribers map[EventType][]Subscriber
	BusMutex    sync.RWMutex
}

func (instance *EventBus) Subscribe(eventType EventType, subscriber Subscriber) {
	instance.BusMutex.Lock()
	defer instance.BusMutex.Unlock()

	instance.Subscribers[eventType] = append(instance.Subscribers[eventType], subscriber)
}

func (instance *EventBus) UnSubscribe(eventType EventType, subscriberID types.Source) {
	instance.BusMutex.Lock()
	defer instance.BusMutex.Unlock()

	for index, subbscriber := range instance.Subscribers[eventType] {
		if subbscriber.GetID() == subscriberID {
			instance.Subscribers[eventType] = append(instance.Subscribers[eventType][:index], instance.Subscribers[eventType][index+1:]...)
			break
		}
	}
}

func (instance *EventBus) Publish(event Event) {
	instance.BusMutex.RLock()
	defer instance.BusMutex.RUnlock()
	for _, subscriber := range instance.Subscribers[event.Type] {
		go func(subscriber Subscriber) {
			subscriber.HandleEvent(event)
		}(subscriber)
	}
}
