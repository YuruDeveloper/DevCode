package events

import (
	devcodeerror "DevCode/src/DevCodeError"
	"DevCode/src/config"
	"DevCode/src/dto"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

func NewEventBus(config config.EventBusConfig, logger *zap.Logger) (*EventBus, error) {
	pool, err := ants.NewPool(config.PoolSize, ants.WithPreAlloc(true))
	if err != nil {
		return nil, devcodeerror.Wrap(err, devcodeerror.FailCreateEventBus, "Fail Create Ant Pool")
	}
	return &EventBus{
		UserInputEvent:    NewTypedBus[dto.UserRequestData](pool, logger),
		UserDecisionEvent: NewTypedBus[dto.UserDecisionData](pool, logger),

		RequestToolUseEvent: NewTypedBus[dto.ToolUseReportData](pool, logger),
		ToolCallEvent:       NewTypedBus[dto.ToolCallData](pool, logger),
		AcceptToolEvent:     NewTypedBus[dto.ToolCallData](pool, logger),
		ToolRawResultEvent:  NewTypedBus[dto.ToolRawResultData](pool, logger),
		ToolResultEvent:     NewTypedBus[dto.ToolResultData](pool, logger),
		ToolUseReportEvent:  NewTypedBus[dto.ToolUseReportData](pool, logger),

		RequestEnvironmentEvent: NewTypedBus[dto.EnvironmentRequestData](pool, logger),
		UpdateEnvironmentEvent:  NewTypedBus[dto.EnvironmentUpdateData](pool, logger),

		RequestToolListEvent: NewTypedBus[dto.RequestToolListData](pool, logger),
		UpdateToolListEvent:  NewTypedBus[dto.ToolListUpdateData](pool, logger),

		StreamStartEvent:    NewTypedBus[dto.StreamStartData](pool, logger),
		StreamChunkEvent:    NewTypedBus[dto.StreamChunkData](pool, logger),
		StreamCompleteEvent: NewTypedBus[dto.StreamCompleteData](pool, logger),
		StreamErrorEvent:    NewTypedBus[dto.StreamErrorData](pool, logger),
		StreamCancelEvent:   NewTypedBus[dto.StreamCancelData](pool, logger),

		StreamChunkParsedEvent:      NewTypedBus[dto.ParsedChunkData](pool, logger),
		StreamChunkParsedErrorEvent: NewTypedBus[dto.ParsedChunkErrorData](pool, logger),
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
