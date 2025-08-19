package service

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
)

func NewToolService(bus *events.EventBus) *ToolService {
	service := &ToolService{
		Bus:     bus,
		Allowed: viper.GetStringSlice("tool.allowed"),
	}
	bus.Subscribe(events.ToolCallEvent, service)
	bus.Subscribe(events.ToolRawResultEvent, service)
	bus.Subscribe(events.UserDecisionEvent, service)
	return service
}

type ToolService struct {
	Bus            *events.EventBus
	Allowed        []string
	ToolCallBuffer map[string]types.ToolCallData
}

func (instance *ToolService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.ToolCallEvent:
		instance.ProcessToolCall(event.Data.(types.ToolCallData))
	case events.ToolRawResultEvent:
		instance.ProcessToolResult(event.Data.(types.ToolRawResultData))
	case events.UserDecisionEvent:
		instance.ProcessUserDecision(event.Data.(types.UserDecisionData))
	}
}

func (instance *ToolService) GetID() types.Source {
	return types.ToolService
}

func (instance *ToolService) ProcessUserDecision(data types.UserDecisionData) {
	if callData , exist := instance.ToolCallBuffer[data.RequestUUID.String() + data.ToolCall.String()]; exist {
		if data.Aceept {
			PublishEvent(instance.Bus,events.AcceptToolEvent,callData,types.ToolService)
		} else {
			var builder strings.Builder
			builder.WriteString("<tool_use_error>\n")
			builder.WriteString("User Reject Tool Use\n ")
			builder.WriteString("</tool_use_error>\n")
			PublishEvent(instance.Bus,events.ToolResultEvent,types.ToolResultData{
				RequestUUID: data.RequestUUID,
				ToolCall: data.ToolCall,
				ToolResult: builder.String(),
			},types.ToolService)
		}
		delete(instance.ToolCallBuffer,data.RequestUUID.String() + data.ToolCall.String())
	}
}

func (instance *ToolService) ProcessToolResult(data types.ToolRawResultData) {
	var builder strings.Builder
	if data.Error != nil {
		builder.WriteString("<tool_use_error>\n")
		builder.WriteString(data.Error.Error() + "\n")
		builder.WriteString("</tool_use_error>\n")
		PublishEvent(instance.Bus, events.ToolResultEvent, types.ToolResultData{
			RequestUUID: data.RequestUUID,
			ToolCall:    data.ToolCall,
			ToolResult:  builder.String(),
		}, types.ToolService)
		PublishEvent(instance.Bus,events.ToolUseReportEvent,types.ToolUseReportData {
			RequestUUID: data.RequestUUID,
			ToolCall: data.ToolCall,
			ToolInfo: "",
			ToolStatus: types.Error,
		},types.ToolService)
		return
	}
	builder.WriteString("<result>\n")
	for _, content := range data.Result.Content {
		builder.WriteString(content.(*mcp.TextContent).Text + "\n")
	}
	builder.WriteString("</result>\n")
	PublishEvent(instance.Bus, events.ToolResultEvent, types.ToolResultData{
		RequestUUID: data.RequestUUID,
		ToolCall:    data.ToolCall,
		ToolResult:  builder.String(),
	}, types.ToolService)
	PublishEvent(instance.Bus,events.ToolUseReportEvent,types.ToolUseReportData {
		RequestUUID: data.RequestUUID,
		ToolCall: data.ToolCall,
		ToolInfo: "",
		ToolStatus: types.Success,
	},types.ToolService)
}

func (instance *ToolService) ProcessToolCall(data types.ToolCallData) {
	for _, Allowed := range instance.Allowed {
		if data.ToolName == Allowed {
			PublishEvent(instance.Bus, events.AcceptToolEvent, data, types.ToolService)
			PublishEvent(instance.Bus, events.ToolUseReportEvent, types.ToolUseReportData{
				RequestUUID: data.RequestUUID,
				ToolCall:    data.ToolCall,
				ToolInfo:    instance.ToolInfo(data.ToolName, data.Parameters),
				ToolStatus:  types.Call,
			}, types.ToolService)
			return
		}
	}
	if instance.ToolCallBuffer == nil {
		instance.ToolCallBuffer = make(map[string]types.ToolCallData)
	}
	instance.ToolCallBuffer[data.RequestUUID.String() + data.ToolCall.String()] = data
	PublishEvent(instance.Bus, events.RequestToolUseEvent, types.ToolUseReportData{
		RequestUUID: data.RequestUUID,
		ToolCall:    data.ToolCall,
		ToolInfo:    instance.ToolInfo(data.ToolName, data.Parameters),
		ToolStatus:  types.Call,
	}, types.ToolService)
}

func (instance *ToolService) ToolInfo(name string, parameters map[string]any) string {
	switch name {
	case "Read":
		return name + " (" + parameters["file_path"].(string) + ")"
	}
	return ""
}
