package events

import (
	"UniCode/src/types"
	"time"
)

type EventType int

const (
	UserInputEvent = EventType(iota + 1)
	ToolCallEvent
	AcceptToolEvent
	RequestEnvironmentEvent
	UpdateEnvironmentEvent
	RequestToolListEvent
	UpdateToolListEvent
	StreamStartEvent
	StreamChunkEvent
	StreamCompleteEvent
	StreamErrorEvent
	StreamCancelEvent
	RequestToolUseEvent
	StreamChunkParsedEvent
	StreamChunkParsedErrorEvent
	ToolRawResultEvent
	ToolResultEvent
	ToolUseReportEvent
	UserDecisionEvent
)

type Event struct {
	Type      EventType
	Data      any
	Timestamp time.Time
	Source    types.Source
}

type Subscriber interface {
	HandleEvent(event Event)
	GetID() types.Source
}
