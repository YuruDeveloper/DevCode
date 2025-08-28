package events

import (
	"DevCode/src/constants"
	"fmt"
	"time"
)

type EventType uint

func (instance EventType) String() string {
	switch instance {
	case UserInputEvent:
		return "UserInputEvent"
	case ToolCallEvent:
		return "ToolCallEvent"
	case AcceptToolEvent:
		return "AcceptToolEvent"
	case RequestEnvironmentEvent:
		return "RequestEnvironmentEvent"
	case UpdateEnvironmentEvent:
		return "UpdateEnvironmentEvent"
	case StreamStartEvent:
		return "StreamStartEvent"
	case StreamChunkEvent:
		return "StreamChunkEvent"
	case StreamCompleteEvent:
		return "StreamCompleteEvent"
	case StreamErrorEvent:
		return "StreamErrorEvent"
	case StreamCancelEvent:
		return "StreamCancelEvent"
	case StreamChunkParsedEvent:
		return "StreamChunkParsedEvent"
	case StreamChunkParsedErrorEvent:
		return "StreamChunkParsedErrorEvent"
	case UserDecisionEvent:
		return "UserDecisionEvent"
	case RequestToolUseEvent:
		return "RequestToolUseEvent"
	case ToolRawResultEvent:
		return "ToolRawResultEvent"
	case ToolResultEvent:
		return "ToolResultEvent"
	case ToolUseReportEvent:
		return "ToolUseReportEvent"
	default:
		return fmt.Sprintf("EventType(%d)", int(instance))
	}
}

const (
	UserInputEvent = EventType(iota + 1)
	UserDecisionEvent

	RequestToolUseEvent
	ToolCallEvent
	AcceptToolEvent
	ToolRawResultEvent
	ToolResultEvent
	ToolUseReportEvent

	RequestEnvironmentEvent
	UpdateEnvironmentEvent

	RequestToolListEvent
	UpdateToolListEvent

	StreamStartEvent
	StreamChunkEvent
	StreamCompleteEvent
	StreamErrorEvent
	StreamCancelEvent

	StreamChunkParsedEvent
	StreamChunkParsedErrorEvent
)

type Event struct {
	Type      EventType
	Data      any
	Timestamp time.Time
	Source    constants.Source
}

type Subscriber interface {
	HandleEvent(event Event)
	GetID() constants.Source
}

type Bus interface {
	Subscribe(eventType EventType, subscriber Subscriber)
	UnSubscribe(eventType EventType, subscriberID constants.Source)
	Publish(event Event)
}
