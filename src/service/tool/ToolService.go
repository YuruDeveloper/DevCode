package tool

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/service"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func NewToolService(bus *events.EventBus, logger *zap.Logger) *ToolService {
	service := &ToolService{
		bus:     bus,
		allowed: viper.GetStringSlice("tool.allowed"),
		logger:  logger,
	}
	bus.Subscribe(events.ToolCallEvent, service)
	bus.Subscribe(events.ToolRawResultEvent, service)
	bus.Subscribe(events.UserDecisionEvent, service)
	return service
}

type ToolService struct {
	bus            *events.EventBus
	allowed        []string
	toolCallBuffer map[uuid.UUID]dto.ToolCallData
	logger         *zap.Logger
}

func (instance *ToolService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.ToolCallEvent:
		instance.ProcessToolCall(event.Data.(dto.ToolCallData))
	case events.ToolRawResultEvent:
		instance.ProcessToolResult(event.Data.(dto.ToolRawResultData))
	case events.UserDecisionEvent:
		instance.ProcessUserDecision(event.Data.(dto.UserDecisionData))
	}
}

func (instance *ToolService) GetID() constants.Source {
	return constants.ToolService
}

func (instance *ToolService) ProcessUserDecision(data dto.UserDecisionData) {
	if callData, exist := instance.toolCallBuffer[data.ToolCallUUID]; exist {
		if data.Accept {
			service.PublishEvent(instance.bus, events.AcceptToolEvent, callData, constants.ToolService)
		} else {
			var builder strings.Builder
			builder.WriteString("<tool_use_error>\n")
			builder.WriteString("User Reject Tool Use\n ")
			builder.WriteString("</tool_use_error>\n")
			service.PublishEvent(instance.bus, events.ToolResultEvent, dto.ToolResultData{
				RequestUUID: data.RequestUUID,
				ToolCall:    data.ToolCallUUID,
				ToolResult:  builder.String(),
			}, constants.ToolService)
		}
		delete(instance.toolCallBuffer, data.ToolCallUUID)
	}
}

func (instance *ToolService) ProcessToolResult(data dto.ToolRawResultData) {
	var builder strings.Builder
	if data.Result.IsError  {
		builder.WriteString("<tool_use_error>\n")
		builder.WriteString(data.Result.Content[0].(*mcp.TextContent).Text + "\n")
		builder.WriteString("</tool_use_error>\n")
		service.PublishEvent(instance.bus, events.ToolResultEvent, dto.ToolResultData{
			RequestUUID: data.RequestUUID,
			ToolCall:    data.ToolCall,
			ToolResult:  builder.String(),
		}, constants.ToolService)
		service.PublishEvent(instance.bus, events.ToolUseReportEvent, dto.ToolUseReportData{
			RequestUUID: data.RequestUUID,
			ToolCall:    data.ToolCall,
			ToolInfo:    "",
			ToolStatus:  constants.Error,
		}, constants.ToolService)
		return
	}
	builder.WriteString("<result>\n")
	for _, content := range data.Result.Content {
		builder.WriteString(content.(*mcp.TextContent).Text + "\n")
	}
	builder.WriteString("</result>\n")
	service.PublishEvent(instance.bus, events.ToolResultEvent, dto.ToolResultData{
		RequestUUID: data.RequestUUID,
		ToolCall:    data.ToolCall,
		ToolResult:  builder.String(),
	}, constants.ToolService)
	service.PublishEvent(instance.bus, events.ToolUseReportEvent, dto.ToolUseReportData{
		RequestUUID: data.RequestUUID,
		ToolCall:    data.ToolCall,
		ToolInfo:    "",
		ToolStatus:  constants.Success,
	}, constants.ToolService)
}

func (instance *ToolService) ProcessToolCall(data dto.ToolCallData) {
	service.PublishEvent(instance.bus, events.ToolUseReportEvent, dto.ToolUseReportData{
		RequestUUID: data.RequestUUID,
		ToolCall:    data.ToolCallUUID,
		ToolInfo:    instance.ToolInfo(data.ToolName, data.Parameters),
		ToolStatus:  constants.Call,
	}, constants.ToolService)
	for _, allowed := range instance.allowed {
		if data.ToolName == allowed {
			service.PublishEvent(instance.bus, events.AcceptToolEvent, data, constants.ToolService)
			return
		}
	}
	if instance.toolCallBuffer == nil {
		instance.toolCallBuffer = make(map[uuid.UUID]dto.ToolCallData)
	}
	instance.toolCallBuffer[data.ToolCallUUID] = data
	service.PublishEvent(instance.bus, events.RequestToolUseEvent, dto.ToolUseReportData{
		RequestUUID: data.RequestUUID,
		ToolCall:    data.ToolCallUUID,
		ToolInfo:    instance.ToolInfo(data.ToolName, data.Parameters),
		ToolStatus:  constants.Call,
	}, constants.ToolService)
}

func (instance *ToolService) ToolInfo(name string, parameters map[string]any) string {
	switch name {
	case "Read":
		return name + " (" + parameters["file_path"].(string) + ")"
	}
	return ""
}
