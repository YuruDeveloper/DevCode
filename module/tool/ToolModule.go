package tool

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
	"strings"
	"time"
)

func NewToolModule(bus *events.EventBus, config config.ToolServiceConfig, logger *zap.Logger) *ToolModule {
	module := &ToolModule{
		bus:            bus,
		allowed:        config.Allowed,
		logger:         logger,
		toolCallBuffer: make(map[types.ToolCallID]dto.ToolCallData),
	}
	module.Subscribe()
	return module
}

type ToolModule struct {
	bus            *events.EventBus
	allowed        []string
	toolCallBuffer map[types.ToolCallID]dto.ToolCallData
	logger         *zap.Logger
}

func (instance *ToolModule) Subscribe() {
	events.Subscribe(instance.bus, instance.bus.ToolCallEvent, constants.ToolModule, func(event events.Event[dto.ToolCallData]) {
		instance.ProcessToolCall(event.Data)
	})
	events.Subscribe(instance.bus, instance.bus.ToolRawResultEvent, constants.ToolModule, func(event events.Event[dto.ToolRawResultData]) {
		instance.ProcessToolResult(event.Data)
	})
	events.Subscribe(instance.bus, instance.bus.UserDecisionEvent, constants.ToolModule, func(event events.Event[dto.UserDecisionData]) {
		instance.ProcessUserDecision(event.Data)
	})
}

func (instance *ToolModule) ProcessUserDecision(data dto.UserDecisionData) {

	if callData, exist := instance.toolCallBuffer[data.ToolCallID]; exist {
		if data.Accept {
			events.Publish(instance.bus, instance.bus.AcceptToolEvent, events.Event[dto.ToolCallData]{
				Data:      callData,
				TimeStamp: time.Now(),
				Source:    constants.ToolModule,
			})
		} else {
			var builder strings.Builder
			builder.WriteString("<tool_use_error>\n")
			builder.WriteString("User Reject Tool Use\n ")
			builder.WriteString("</tool_use_error>\n")
			events.Publish(instance.bus, instance.bus.ToolResultEvent,
				events.Event[dto.ToolResultData]{
					Data: dto.ToolResultData{
						RequestID:  data.RequestID,
						ToolCallID: data.ToolCallID,
						ToolResult: builder.String(),
					},
					TimeStamp: time.Now(),
					Source:    constants.ToolModule,
				})
			events.Publish(instance.bus, instance.bus.ToolUseReportEvent, events.Event[dto.ToolUseReportData]{
				Data: dto.ToolUseReportData{
					RequestID:  data.RequestID,
					ToolCallID: data.ToolCallID,
					ToolInfo:   "",
					ToolStatus: constants.Error,
				},
				TimeStamp: time.Now(),
				Source:    constants.ToolModule,
			})
		}
		delete(instance.toolCallBuffer, data.ToolCallID)
	} else {
		instance.logger.Error("Tool call not found in buffer",
			zap.String("tool_call_uuid", data.ToolCallID.String()))
	}
}

func (instance *ToolModule) ProcessToolResult(data dto.ToolRawResultData) {

	var builder strings.Builder
	if data.Result.IsError {
		errorText := data.Result.Content[0].(*mcp.TextContent).Text
		instance.logger.Error("Tool execution failed",
			zap.String("tool_call_uuid", data.ToolCallID.String()),
			zap.String("error", errorText))

		builder.WriteString("<tool_use_error>\n")
		builder.WriteString(errorText + "\n")
		builder.WriteString("</tool_use_error>\n")
		events.Publish(instance.bus, instance.bus.ToolResultEvent,
			events.Event[dto.ToolResultData]{
				Data: dto.ToolResultData{
					RequestID:  data.RequestID,
					ToolCallID: data.ToolCallID,
					ToolResult: builder.String(),
				},
				TimeStamp: time.Now(),
				Source:    constants.ToolModule,
			})
		events.Publish(instance.bus, instance.bus.ToolUseReportEvent, events.Event[dto.ToolUseReportData]{
			Data: dto.ToolUseReportData{
				RequestID:  data.RequestID,
				ToolCallID: data.ToolCallID,
				ToolInfo:   "",
				ToolStatus: constants.Error,
			},
			TimeStamp: time.Now(),
			Source:    constants.ToolModule,
		})
		return
	}

	builder.WriteString("<result>\n")
	for _, content := range data.Result.Content {
		builder.WriteString(content.(*mcp.TextContent).Text + "\n")
	}
	builder.WriteString("</result>\n")
	events.Publish(instance.bus, instance.bus.ToolResultEvent,
		events.Event[dto.ToolResultData]{
			Data: dto.ToolResultData{
				RequestID:  data.RequestID,
				ToolCallID: data.ToolCallID,
				ToolResult: builder.String(),
			},
			TimeStamp: time.Now(),
			Source:    constants.ToolModule,
		})
	events.Publish(instance.bus, instance.bus.ToolUseReportEvent, events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  data.RequestID,
			ToolCallID: data.ToolCallID,
			ToolInfo:   "",
			ToolStatus: constants.Success,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	})
}

func (instance *ToolModule) ProcessToolCall(data dto.ToolCallData) {

	events.Publish(instance.bus, instance.bus.ToolUseReportEvent, events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  data.RequestID,
			ToolCallID: data.ToolCallID,
			ToolInfo:   instance.ToolInfo(data.ToolName, data.Parameters),
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	})
	for _, allowed := range instance.allowed {
		if data.ToolName == allowed {
			events.Publish(instance.bus, instance.bus.AcceptToolEvent, events.Event[dto.ToolCallData]{
				Data:      data,
				TimeStamp: time.Now(),
				Source:    constants.ToolModule,
			})
			return
		}
	}

	instance.toolCallBuffer[data.ToolCallID] = data
	events.Publish(instance.bus, instance.bus.RequestToolUseEvent, events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  data.RequestID,
			ToolCallID: data.ToolCallID,
			ToolInfo:   instance.ToolInfo(data.ToolName, data.Parameters),
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	})
}

func (instance *ToolModule) ToolInfo(name string, parameters map[string]any) string {
	switch name {
	case "Read":
		if filePath, ok := parameters["file_path"].(string); ok {
			return fmt.Sprintf("%s (%s)", name, filePath)
		}
		return name
	case "List":
		if path, ok := parameters["path"].(string); ok {
			return fmt.Sprintf("%s (%s)", name, path)
		}
		return name
	}
	return name
}
