package tool

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/types"
	"fmt"
	"strings"
	"time"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

func NewToolService(bus *events.EventBus, config config.ToolServiceConfig, logger *zap.Logger) *ToolService {
	service := &ToolService{
		bus:     bus,
		allowed: config.Allowed,
		logger:  logger,
		toolCallBuffer: make(map[types.ToolCallID]dto.ToolCallData),
	}
	service.Subscribe()
	return service
}

type ToolService struct {
	bus            *events.EventBus
	allowed        []string
	toolCallBuffer map[types.ToolCallID]dto.ToolCallData
	logger         *zap.Logger
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

	if callData, exist := instance.toolCallBuffer[data.ToolCallID]; exist {
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
						RequestID:  data.RequestID,
						ToolCallID: data.ToolCallID,
						ToolResult:   builder.String(),
					},
					TimeStamp: time.Now(),
					Source:    constants.ToolService,
				})
			instance.bus.ToolUseReportEvent.Publish(events.Event[dto.ToolUseReportData]{
				Data: dto.ToolUseReportData{
					RequestID:  data.RequestID,
					ToolCallID: data.ToolCallID,
					ToolInfo:     "",
					ToolStatus:   constants.Error,
				},
				TimeStamp: time.Now(),
				Source:    constants.ToolService,
			})
		}
		delete(instance.toolCallBuffer, data.ToolCallID)
	} else {
		instance.logger.Error("Tool call not found in buffer",
			zap.String("tool_call_uuid", data.ToolCallID.String()))
	}
}

func (instance *ToolService) ProcessToolResult(data dto.ToolRawResultData) {

	var builder strings.Builder
	if data.Result.IsError {
		errorText := data.Result.Content[0].(*mcp.TextContent).Text
		instance.logger.Error("Tool execution failed",
			zap.String("tool_call_uuid", data.ToolCallID.String()),
			zap.String("error", errorText))

		builder.WriteString("<tool_use_error>\n")
		builder.WriteString(errorText + "\n")
		builder.WriteString("</tool_use_error>\n")
		instance.bus.ToolResultEvent.Publish(
			events.Event[dto.ToolResultData]{
				Data: dto.ToolResultData{
					RequestID:  data.RequestID,
					ToolCallID: data.ToolCallID,
					ToolResult:   builder.String(),
				},
				TimeStamp: time.Now(),
				Source:    constants.ToolService,
			})
		instance.bus.ToolUseReportEvent.Publish(events.Event[dto.ToolUseReportData]{
			Data: dto.ToolUseReportData{
				RequestID:  data.RequestID,
				ToolCallID: data.ToolCallID,
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
				RequestID:  data.RequestID,
				ToolCallID: data.ToolCallID,
				ToolResult:   builder.String(),
			},
			TimeStamp: time.Now(),
			Source:    constants.ToolService,
		})
	instance.bus.ToolUseReportEvent.Publish(events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  data.RequestID,
			ToolCallID: data.ToolCallID,
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
			RequestID:  data.RequestID,
			ToolCallID: data.ToolCallID,
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



	instance.toolCallBuffer[data.ToolCallID] = data
	instance.bus.RequestToolUseEvent.Publish(events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  data.RequestID,
			ToolCallID: data.ToolCallID,
			ToolInfo:     instance.ToolInfo(data.ToolName, data.Parameters),
			ToolStatus:   constants.Call,
		},
	})
}

func (instance *ToolService) ToolInfo(name string, parameters map[string]any) string {
	switch name {
	case "Read":
		if filePath , ok := parameters["file_path"].(string) ; ok {
			return fmt.Sprintf("%s (%s)",name,filePath)
		}
		return name
	case "List":
		if path , ok := parameters["path"].(string) ; ok {
			return fmt.Sprintf("%s (%s)",name,path)
		}
		return name
	}
	return name
}
