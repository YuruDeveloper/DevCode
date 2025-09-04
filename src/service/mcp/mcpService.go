package mcp

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/tools/ls"
	"DevCode/src/tools/read"
	"DevCode/src/types"
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type McpService struct {
	client        *mcp.Client
	clientSession *mcp.ClientSession
	toolServer    *mcp.Server
	bus           *events.EventBus
	ctx           context.Context
}

func NewMcpService(bus *events.EventBus, config config.McpServiceConfig) *McpService {

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: config.Name, Version: config.Version}, nil)

	implementation := &mcp.Implementation{
		Name:    config.ServerName,
		Version: config.ServerVersion,
	}
	mcpServer := mcp.NewServer(implementation, nil)

	service := &McpService{
		client:     mcpClient,
		bus:        bus,
		toolServer: mcpServer,
		ctx:        context.Background(),
	}

	serverTran, clientTrans := mcp.NewInMemoryTransports()

	service.InitTools()

	go func() {
		if err := service.toolServer.Run(service.ctx, serverTran); err != nil {
		}
	}()

	service.clientSession, _ = service.client.Connect(service.ctx, clientTrans)

	service.Subscribe()
	return service
}

func (instance *McpService) Subscribe() {
	instance.bus.RequestToolListEvent.Subscribe(constants.McpService, func(event events.Event[dto.RequestToolListData]) {
		instance.PublishToolList()
	})
	instance.bus.AcceptToolEvent.Subscribe(constants.McpService, func(event events.Event[dto.ToolCallData]) {
		instance.ToolCall(event.Data)
	})
}

func (instance *McpService) InitTools() {
	InsertTool(instance, &read.Tool{})
	InsertTool(instance, &ls.Tool{})
}

func InsertTool[T any](server *McpService, tool types.Tool[T]) {

	mcpTool := &mcp.Tool{
		Name:        tool.Name(),
		Description: tool.Description(),
	}
	mcp.AddTool(server.toolServer, mcpTool, tool.Handler())
}

func (instance *McpService) ToolCall(data dto.ToolCallData) {

	params := &mcp.CallToolParams{
		Name:      data.ToolName,
		Arguments: data.Parameters,
	}

	result, err := instance.clientSession.CallTool(instance.ctx, params)

	if err != nil {
		instance.bus.ToolRawResultEvent.Publish(events.Event[dto.ToolRawResultData]{
			Data: dto.ToolRawResultData{
				RequestUUID:  data.RequestUUID,
				ToolCallUUID: data.ToolCallUUID,
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
			Source:    constants.McpService,
		})
		return
	}
	instance.bus.ToolRawResultEvent.Publish(events.Event[dto.ToolRawResultData]{
		Data: dto.ToolRawResultData{
			RequestUUID:  data.RequestUUID,
			ToolCallUUID: data.ToolCallUUID,
			Result:       result,
		},
		TimeStamp: time.Now(),
		Source:    constants.McpService,
	})
}

func (instance *McpService) PublishToolList() {
	mcpToolList := make([]*mcp.Tool, 0, 10)
	for tool := range instance.clientSession.Tools(instance.ctx, nil) {
		mcpToolList = append(mcpToolList, tool)
	}
	instance.bus.UpdateToolListEvent.Publish(events.Event[dto.ToolListUpdateData]{
		Data: dto.ToolListUpdateData{
			List: mcpToolList,
		},
		TimeStamp: time.Now(),
		Source:    constants.McpService,
	})
}

func (instance *McpService) GetID() constants.Source {
	return constants.McpService
}
