 package mcp

import (
	devcodeerror "DevCode/DevCodeError"
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/tools/list"
	"DevCode/tools/read"
	"DevCode/types"
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

type McpModule struct {
	client        *mcp.Client
	clientSession *mcp.ClientSession
	toolServer    *mcp.Server
	bus           *events.EventBus
	ctx           context.Context
	logger        *zap.Logger
}

func NewMcpModule(bus *events.EventBus, config config.McpServiceConfig, logger *zap.Logger) *McpModule {

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: config.Name, Version: config.Version}, nil)

	implementation := &mcp.Implementation{
		Name:    config.ServerName,
		Version: config.ServerVersion,
	}
	mcpServer := mcp.NewServer(implementation, nil)

	module := &McpModule{
		client:     mcpClient,
		bus:        bus,
		toolServer: mcpServer,
		ctx:        context.Background(),
		logger:     logger,
	}

	serverTran, clientTrans := mcp.NewInMemoryTransports()

	module.InitTools()

	go func() {
		if err := module.toolServer.Run(module.ctx, serverTran); err != nil {
			module.logger.Error("", zap.Error(devcodeerror.Wrap(
				err,
				devcodeerror.FailRunMcpServer,
				"Fail Run MCP Server",
			)))
		}
	}()

	module.clientSession, _ = module.client.Connect(module.ctx, clientTrans)

	module.Subscribe()
	return module
}

func (instance *McpModule) Subscribe() {
	events.Subscribe(instance.bus,instance.bus.RequestToolListEvent,constants.McpModule,func(event events.Event[dto.RequestToolListData]) {
		instance.PublishToolList()
	})
	events.Subscribe(instance.bus,instance.bus.AcceptToolEvent,constants.McpModule,func(event events.Event[dto.ToolCallData]) {
		instance.ToolCall(event.Data)
	})
}

func (instance *McpModule) InitTools() {
	InsertTool(instance, &read.Tool{})
	InsertTool(instance, &list.Tool{})
}

func InsertTool[T any](server *McpModule, tool types.Tool[T]) {

	mcpTool := &mcp.Tool{
		Name:        tool.Name(),
		Description: tool.Description(),
	}
	mcp.AddTool(server.toolServer, mcpTool, tool.Handler())

}

func (instance *McpModule) ToolCall(data dto.ToolCallData) {

	params := &mcp.CallToolParams{
		Name:      data.ToolName,
		Arguments: data.Parameters,
	}

	result, err := instance.clientSession.CallTool(instance.ctx, params)

	if err != nil {
		instance.logger.Error("도구 호출 실패",
			zap.String("toolName", data.ToolName),
			zap.String("requestUUID", data.RequestID.String()),
			zap.String("toolCallUUID", data.ToolCallID.String()),
			zap.Error(err))
		events.Publish(instance.bus,instance.bus.ToolRawResultEvent,events.Event[dto.ToolRawResultData]{
			Data: dto.ToolRawResultData{
				RequestID:  data.RequestID,
				ToolCallID: data.ToolCallID,
				Result: &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: fmt.Sprintf("Tool Call Error : %v", err),
						},
					},
				},
			},
			TimeStamp: time.Now(),
			Source:    constants.McpModule,
		})
		return
	}
	events.Publish(instance.bus,instance.bus.ToolRawResultEvent,events.Event[dto.ToolRawResultData]{
		Data: dto.ToolRawResultData{
			RequestID:  data.RequestID,
			ToolCallID: data.ToolCallID,
			Result:     result,
		},
		TimeStamp: time.Now(),
		Source:    constants.McpModule,
	})
}

func (instance *McpModule) PublishToolList() {

	mcpToolList := make([]*mcp.Tool, 0, 10)
	for tool := range instance.clientSession.Tools(instance.ctx, nil) {
		mcpToolList = append(mcpToolList, tool)
	}
	events.Publish(instance.bus,instance.bus.UpdateToolListEvent,events.Event[dto.ToolListUpdateData]{
		Data: dto.ToolListUpdateData{
			List: mcpToolList,
		},
		TimeStamp: time.Now(),
		Source:    constants.McpModule,
	})
}
