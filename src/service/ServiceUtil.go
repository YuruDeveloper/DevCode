package service

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"time"
)

func PublishEvent(bus *events.EventBus, eventType events.EventType, data any, source types.Source) {
	bus.Publish(
		events.Event{
			Type:      eventType,
			Data:      data,
			Timestamp: time.Now(),
			Source:    source,
		},
	)
}
