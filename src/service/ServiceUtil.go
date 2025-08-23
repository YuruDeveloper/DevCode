package service

import (
	"UniCode/src/constants"
	"UniCode/src/events"
	"time"
)

func PublishEvent(bus events.Bus, eventType events.EventType, data any, source constants.Source) {
	bus.Publish(
		events.Event{
			Type:      eventType,
			Data:      data,
			Timestamp: time.Now(),
			Source:    source,
		},
	)
}
