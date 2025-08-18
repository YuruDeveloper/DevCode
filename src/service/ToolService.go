package service

import (
	"UniCode/src/events"
	"UniCode/src/types"

	"github.com/spf13/viper"
)

func NewToolService(bus *events.EventBus) *ToolService {
	service := &ToolService{
		Bus:     bus,
		Allowed: viper.GetStringSlice("tool.allowed"),
	}
	bus.Subscribe(events.ToolCallEvent, service)
	return service
}

type ToolService struct {
	Bus     *events.EventBus
	Allowed []string
}

func (instance *ToolService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.ToolCallEvent:
		instance.ProcessToolCall(event.Data.(types.ToolCallData))
	} 
}

func (instance *ToolService) GetID() types.Source {
	return types.ToolService
}

func (instance *ToolService) ProcessToolCall(data types.ToolCallData) {
	for _, Allowed := range instance.Allowed {
		if data.ToolName == Allowed {
			PublishEvent(instance.Bus, events.AcceptToolEvent, data, types.ToolService)
			PublishEvent(instance.Bus, events.RequesetToolUseEvent, types.RequestToolUseData{
				RequestUUID: data.RequestUUID,
				ToolName:    data.ToolName,
				Paramters:   data.Paramters,
				AllowNeed:   false,
			}, types.ToolService)
		}
	}
}
