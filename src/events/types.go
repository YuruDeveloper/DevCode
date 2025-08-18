package events

import (
	"UniCode/src/types"
	"time"
)

type EventType int

const (
	UserInputEvent = iota + 1
	ToolCallEvent
	ErrorEvent
	AcceptToolEvent
	RequestEnvionmentvent
	UpdateEnvionmentEvent
	RequestToolListEvent
	UpdateToolListEvent
	StreamStartEvent
	StreamChunkEvent
	StreamCompleteEvent
	StreamErrorEvent
	StreamCancelEvent
	ToolErrorEvent
	ToolCompleteEvent
	RequesetToolUseEvent
	StreamChunkParsedEvent
	StreamChunkParsedErrorEvent
	ToolResultEvent
	AcceptEvent
	RejectEvent
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
