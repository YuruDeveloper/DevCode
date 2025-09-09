package events

import (
	devcodeerror "DevCode/DevCodeError"
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"time"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

func NewEventBus(config config.EventBusConfig, logger *zap.Logger) (*EventBus, error) {
	pool, err := ants.NewPool(config.PoolSize, ants.WithPreAlloc(true))
	if err != nil {
		return nil, devcodeerror.Wrap(err, devcodeerror.FailCreateEventBus, "Fail Create Ant Pool")
	}
	bus := &EventBus{
		UserInputEvent:        NewTypedBus[dto.UserRequestData](),
		UserDecisionEvent:     NewTypedBus[dto.UserDecisionData](),
		UpdateUserStatusEvent: NewTypedBus[dto.UpdateUserStatusData](),
		UpdateViewEvent:       NewTypedBus[dto.UpdateViewData](),

		RequestToolUseEvent: NewTypedBus[dto.ToolUseReportData](),
		ToolCallEvent:       NewTypedBus[dto.ToolCallData](),
		AcceptToolEvent:     NewTypedBus[dto.ToolCallData](),
		ToolRawResultEvent:  NewTypedBus[dto.ToolRawResultData](),
		ToolResultEvent:     NewTypedBus[dto.ToolResultData](),
		ToolUseReportEvent:  NewTypedBus[dto.ToolUseReportData](),

		RequestEnvironmentEvent: NewTypedBus[dto.EnvironmentRequestData](),
		UpdateEnvironmentEvent:  NewTypedBus[dto.EnvironmentUpdateData](),

		RequestToolListEvent: NewTypedBus[dto.RequestToolListData](),
		UpdateToolListEvent:  NewTypedBus[dto.ToolListUpdateData](),

		StreamStartEvent:    NewTypedBus[dto.StreamStartData](),
		StreamChunkEvent:    NewTypedBus[dto.StreamChunkData](),
		StreamCompleteEvent: NewTypedBus[dto.StreamCompleteData](),
		StreamErrorEvent:    NewTypedBus[dto.StreamErrorData](),
		StreamCancelEvent:   NewTypedBus[dto.StreamCancelData](),

		StreamChunkParsedEvent:      NewTypedBus[dto.ParsedChunkData](),
		StreamChunkParsedErrorEvent: NewTypedBus[dto.ParsedChunkErrorData](),

		RagnarokEvent: NewTypedBus[dto.RagnarokData](),

		logger: logger,
		pool:   pool,
	}
	return bus, nil
}

type EventBus struct {
	UserInputEvent        *TypedBus[dto.UserRequestData]
	UserDecisionEvent     *TypedBus[dto.UserDecisionData]
	UpdateUserStatusEvent *TypedBus[dto.UpdateUserStatusData]
	UpdateViewEvent       *TypedBus[dto.UpdateViewData]

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

	RagnarokEvent *TypedBus[dto.RagnarokData]

	logger *zap.Logger
	pool   *ants.Pool
}

func (instance *EventBus) Ragnarok() {
	Publish(instance, instance.RagnarokEvent, Event[dto.RagnarokData]{
		Data:      dto.RagnarokData{},
		TimeStamp: time.Now(),
	})
}

func (instance *EventBus) Close() {
	instance.pool.Release()
}

func Subscribe[T any](bus *EventBus, typedBus *TypedBus[T], source constants.Source, handler func(Event[T])) {
	typedBus.handlerMutex.Lock()
	defer typedBus.handlerMutex.Unlock()
	typedBus.handlers[source] = handler
}

func UnSubscribe[T any](bus *EventBus, typedBus *TypedBus[T], source constants.Source) {
	typedBus.handlerMutex.Lock()
	defer typedBus.handlerMutex.Unlock()
	delete(typedBus.handlers, source)
}

func Publish[T any](bus *EventBus, typedBus *TypedBus[T], event Event[T]) {
	typedBus.handlerMutex.RLock()
	copyed := make([]func(Event[T]), 0, len(typedBus.handlers))
	for _, handler := range typedBus.handlers {
		copyed = append(copyed, handler)
	}
	typedBus.handlerMutex.RUnlock()
	for _, handler := range copyed {
		copyedHandler := handler
		bus.pool.Submit(func() {
			defer func() {
				if recover := recover(); recover != nil {
					bus.logger.Error("", zap.Any("recover", recover),
						zap.Error(devcodeerror.Wrap(
							nil,
							devcodeerror.FailHandleEvent,
							"Fail HandleEvent",
						)))
					bus.Ragnarok()
				}
			}()
			copyedHandler(event)
		})
	}
}
