package tool

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
)

func NewToolService(bus *events.EventBus) *ToolService {
	service := &ToolService{
		bus:     bus,
		allowed: viper.GetStringSlice("tool.allowed"),
	}
	service.Subscribe()
	return service
}

type ToolService struct {
	bus            *events.EventBus
	allowed        []string
	toolCallBuffer map[uuid.UUID]dto.ToolCallData
}

func (instance *ToolService) Subscribe() {
	instance.bus.ToolCallEvent.Subscribe(constants.ToolService, func(event events.Event[dto.ToolCallData]) {
		instance.ProcessToolCall(event.Data)
	})
	instance.bus.ToolRawResultEvent.Subscribe(constants.ToolService, func(event events.Event[dto.ToolRawResultData]) {
		instance.ProcessToolResult(event.Data)
	})
	instance.bus.UserDecisionEvent.Subscribe(constants.ToolService, func(event events.Event[dto.UserDecisionData]) {
		instance.ProcessUserDecision(event.Data)
	})
}

func (instance *ToolService) ProcessUserDecision(data dto.UserDecisionData) {
	if callData, exist := instance.toolCallBuffer[data.ToolCallUUID]; exist {
		if data.Accept {
			instance.bus.AcceptToolEvent.Publish(events.Event[dto.ToolCallData]{
				Data:      callData,
				TimeStamp: time.Now(),
				Source:    constants.ToolService,
			})
		} else {
			var builder strings.Builder
			builder.WriteString("<tool_use_error>\n")
			builder.WriteString("User Reject Tool Use\n ")
			builder.WriteString("</tool_use_error>\n")
			instance.bus.ToolResultEvent.Publish(
				events.Event[dto.ToolResultData]{
					Data: dto.ToolResultData{
						RequestUUID:  data.RequestUUID,
						ToolCallUUID: data.ToolCallUUID,
						ToolResult:   builder.String(),
					},
					TimeStamp: time.Now(),
					Source:    constants.ToolService,
				})
			instance.bus.ToolUseReportEvent.Publish(events.Event[dto.ToolUseReportData]{
				Data: dto.ToolUseReportData{
					RequestUUID:  data.RequestUUID,
					ToolCallUUID: data.ToolCallUUID,
					ToolInfo:     "",
					ToolStatus:   constants.Error,
				},
				TimeStamp: time.Now(),
				Source:    constants.ToolService,
			})
		}
		delete(instance.toolCallBuffer, data.ToolCallUUID)
	}
}

func (instance *ToolService) ProcessToolResult(data dto.ToolRawResultData) {
	var builder strings.Builder
	if data.Result.IsError {
		builder.WriteString("<tool_use_error>\n")
		builder.WriteString(data.Result.Content[0].(*mcp.TextContent).Text + "\n")
		builder.WriteString("</tool_use_error>\n")
		instance.bus.ToolResultEvent.Publish(
			events.Event[dto.ToolResultData]{
				Data: dto.ToolResultData{
					RequestUUID:  data.RequestUUID,
					ToolCallUUID: data.ToolCallUUID,
					ToolResult:   builder.String(),
				},
				TimeStamp: time.Now(),
				Source:    constants.ToolService,
			})
		instance.bus.ToolUseReportEvent.Publish(events.Event[dto.ToolUseReportData]{
			Data: dto.ToolUseReportData{
				RequestUUID:  data.RequestUUID,
				ToolCallUUID: data.ToolCallUUID,
				ToolInfo:     "",
				ToolStatus:   constants.Error,
			},
			TimeStamp: time.Now(),
			Source:    constants.ToolService,
		})
		return
	}
	builder.WriteString("<result>\n")
	for _, content := range data.Result.Content {
		builder.WriteString(content.(*mcp.TextContent).Text + "\n")
	}
	builder.WriteString("</result>\n")
	instance.bus.ToolResultEvent.Publish(
		events.Event[dto.ToolResultData]{
			Data: dto.ToolResultData{
				RequestUUID:  data.RequestUUID,
				ToolCallUUID: data.ToolCallUUID,
				ToolResult:   builder.String(),
			},
			TimeStamp: time.Now(),
			Source:    constants.ToolService,
		})
	instance.bus.ToolUseReportEvent.Publish(events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestUUID:  data.RequestUUID,
			ToolCallUUID: data.ToolCallUUID,
			ToolInfo:     "",
			ToolStatus:   constants.Success,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolService,
	})
}

func (instance *ToolService) ProcessToolCall(data dto.ToolCallData) {
	instance.bus.ToolUseReportEvent.Publish(events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestUUID:  data.RequestUUID,
			ToolCallUUID: data.ToolCallUUID,
			ToolInfo:     instance.ToolInfo(data.ToolName, data.Parameters),
			ToolStatus:   constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolService,
	})
	for _, allowed := range instance.allowed {
		if data.ToolName == allowed {
			instance.bus.AcceptToolEvent.Publish(events.Event[dto.ToolCallData]{
				Data:      data,
				TimeStamp: time.Now(),
				Source:    constants.ToolService,
			})
			return
		}
	}
	if instance.toolCallBuffer == nil {
		instance.toolCallBuffer = make(map[uuid.UUID]dto.ToolCallData)
	}
	instance.toolCallBuffer[data.ToolCallUUID] = data
	instance.bus.RequestToolUseEvent.Publish(events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestUUID:  data.RequestUUID,
			ToolCallUUID: data.ToolCallUUID,
			ToolInfo:     instance.ToolInfo(data.ToolName, data.Parameters),
			ToolStatus:   constants.Call,
		},
	})
}

func (instance *ToolService) ToolInfo(name string, parameters map[string]any) string {
	switch name {
	case "Read":
		return name + " (" + parameters["file_path"].(string) + ")"
	case "LS":
		return "List(" + parameters["path"].(string) + ")"
	}
	return ""
}
