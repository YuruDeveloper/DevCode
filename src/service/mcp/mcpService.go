package mcp

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/service"
	"DevCode/src/tools/read"
	"DevCode/src/types"
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
)

type McpService struct {
	client        *mcp.Client
	clientSession *mcp.ClientSession
	toolServer    *mcp.Server
	bus           *events.EventBus
	ctx           context.Context
}

func NewMcpService(bus *events.EventBus) *McpService {

	requireds := []string{"mcp.name", "mcp.version", "server.name", "server.version"}
	data := make([]string,4)
	for index, required := range requireds {
		data[index] = viper.GetString(required)
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: data[0], Version: data[1]}, nil)

	implementation := &mcp.Implementation{
		Name:    data[2],
		Version: data[3],
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

	bus.Subscribe(events.RequestToolListEvent, service)
	bus.Subscribe(events.AcceptToolEvent, service)
	return service
}

func (instance *McpService) InitTools() {
	InsertTool(instance, &read.Tool{})
}

func InsertTool[T any](server *McpService, tool types.Tool[T]) {

	mcpTool := &mcp.Tool{
		Name:        tool.Name(),
		Description: tool.Description(),
	}
	mcp.AddTool(server.toolServer, mcpTool, tool.Handler())
}

func (instance *McpService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.RequestToolListEvent:
		instance.PublishToolList()
	case events.AcceptToolEvent:
		instance.ToolCall(event.Data.(dto.ToolCallData))
	default:
	}
}

func (instance *McpService) ToolCall(data dto.ToolCallData) {

	params := &mcp.CallToolParams{
		Name:      data.ToolName,
		Arguments: data.Parameters,
	}

	result, err := instance.clientSession.CallTool(instance.ctx, params)

	if err != nil {
		service.PublishEvent(instance.bus,events.ToolRawResultEvent,dto.ToolRawResultData{
			RequestUUID: data.RequestUUID,
			ToolCall: data.ToolCallUUID,
			Result: &mcp.CallToolResult {
				IsError: true,
				Content: []mcp.Content {
					&mcp.TextContent{
						Text: fmt.Sprintf("Tool Call Error : %v",err),
					},
				},
			},
		},constants.McpService)
		return
	}
	service.PublishEvent(instance.bus, events.ToolRawResultEvent, dto.ToolRawResultData{
		RequestUUID: data.RequestUUID,
		ToolCall:    data.ToolCallUUID,
		Result:      result,
	}, constants.McpService)
}

func (instance *McpService) PublishToolList() {
	mcpToolList := make([]*mcp.Tool, 0, 10)
	for tool := range instance.clientSession.Tools(instance.ctx, nil) {
		mcpToolList = append(mcpToolList, tool)
	}

	service.PublishEvent(instance.bus, events.UpdateToolListEvent,
		dto.ToolListUpdateData{
			List: mcpToolList,
		},
		constants.McpService)
}

func (instance *McpService) GetID() constants.Source {
	return constants.McpService
}
