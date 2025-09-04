package events

import (
	"DevCode/src/dto"
	"fmt"
	"github.com/panjf2000/ants/v2"
)

func NewEventBus() (*EventBus, error) {
	pool, err := ants.NewPool(10000, ants.WithPreAlloc(true))
	if err != nil {
		return nil, fmt.Errorf("fail to create ants pool : %w", err)
	}
	return &EventBus{
		UserInputEvent:    NewTypedBus[dto.UserRequestData](pool),
		UserDecisionEvent: NewTypedBus[dto.UserDecisionData](pool),

		RequestToolUseEvent: NewTypedBus[dto.ToolUseReportData](pool),
		ToolCallEvent:       NewTypedBus[dto.ToolCallData](pool),
		AcceptToolEvent:     NewTypedBus[dto.ToolCallData](pool),
		ToolRawResultEvent:  NewTypedBus[dto.ToolRawResultData](pool),
		ToolResultEvent:     NewTypedBus[dto.ToolResultData](pool),
		ToolUseReportEvent:  NewTypedBus[dto.ToolUseReportData](pool),

		RequestEnvironmentEvent: NewTypedBus[dto.EnvironmentRequestData](pool),
		UpdateEnvironmentEvent:  NewTypedBus[dto.EnvironmentUpdateData](pool),

		RequestToolListEvent: NewTypedBus[dto.RequestToolListData](pool),
		UpdateToolListEvent:  NewTypedBus[dto.ToolListUpdateData](pool),

		StreamStartEvent:    NewTypedBus[dto.StreamStartData](pool),
		StreamChunkEvent:    NewTypedBus[dto.StreamChunkData](pool),
		StreamCompleteEvent: NewTypedBus[dto.StreamCompleteData](pool),
		StreamErrorEvent:    NewTypedBus[dto.StreamErrorData](pool),
		StreamCancelEvent:   NewTypedBus[dto.StreamCancelData](pool),

		StreamChunkParsedEvent:      NewTypedBus[dto.ParsedChunkData](pool),
		StreamChunkParsedErrorEvent: NewTypedBus[dto.ParsedChunkErrorData](pool),
		pool:                        pool,
	}, nil
}

type EventBus struct {
	UserInputEvent    *TypedBus[dto.UserRequestData]
	UserDecisionEvent *TypedBus[dto.UserDecisionData]

	RequestToolUseEvent *TypedBus[dto.ToolUseReportData]
	ToolCallEvent       *TypedBus[dto.ToolCallData]
	AcceptToolEvent     *TypedBus[dto.ToolCallData]
	ToolRawResultEvent  *TypedBus[dto.ToolRawResultData]
	ToolResultEvent     *TypedBus[dto.ToolResultData]
	ToolUseReportEvent  *TypedBus[dto.ToolUseReportData]

	RequestEnvironmentEvent *TypedBus[dto.EnvironmentRequestData]
	UpdateEnvironmentEvent  *TypedBus[dto.EnvironmentUpdateData]

	RequestToolListEvent *TypedBus[dto.RequestToolListData]
	UpdateToolListEvent  *TypedBus[dto.ToolListUpdateData]

	StreamStartEvent    *TypedBus[dto.StreamStartData]
	StreamChunkEvent    *TypedBus[dto.StreamChunkData]
	StreamCompleteEvent *TypedBus[dto.StreamCompleteData]
	StreamErrorEvent    *TypedBus[dto.StreamErrorData]
	StreamCancelEvent   *TypedBus[dto.StreamCancelData]

	StreamChunkParsedEvent      *TypedBus[dto.ParsedChunkData]
	StreamChunkParsedErrorEvent *TypedBus[dto.ParsedChunkErrorData]

	pool *ants.Pool
}

func (instance *EventBus) Close() {
	instance.UserInputEvent.Close()
	instance.UserDecisionEvent.Close()

	instance.RequestEnvironmentEvent.Close()
	instance.UpdateEnvironmentEvent.Close()

	instance.RequestToolListEvent.Close()
	instance.UpdateToolListEvent.Close()

	instance.RequestToolUseEvent.Close()
	instance.ToolUseReportEvent.Close()
	instance.ToolCallEvent.Close()
	instance.AcceptToolEvent.Close()
	instance.ToolResultEvent.Close()
	instance.ToolRawResultEvent.Close()

	instance.StreamStartEvent.Close()
	instance.StreamChunkEvent.Close()
	instance.StreamCancelEvent.Close()
	instance.StreamCompleteEvent.Close()
	instance.StreamErrorEvent.Close()

	instance.StreamChunkParsedEvent.Close()
	instance.StreamChunkParsedErrorEvent.Close()

	instance.pool.Release()
}
