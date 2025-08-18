package service

import (
	"UniCode/src/events"
	"UniCode/src/tools/read"
	"UniCode/src/types"
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
)

type McpService struct {
	Client        *mcp.Client
	ClientSession *mcp.ClientSession
	ToolServer    *mcp.Server
	Bus           *events.EventBus
	ctx           context.Context
}

func NewMcpService(bus *events.EventBus) *McpService {
	mcpClientName := viper.GetString("mcp.name")
	mcpVersion := viper.GetString("mcp.version")
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: mcpClientName, Version: mcpVersion}, nil)

	mcpServerName := viper.GetString("server.name")
	mcpServerVersion := viper.GetString("server.version")
	implementation := &mcp.Implementation{
		Name:    mcpServerName,
		Version: mcpServerVersion,
	}
	mcpServer := mcp.NewServer(implementation, nil)

	service := &McpService{
		Client:     mcpClient,
		Bus:        bus,
		ToolServer: mcpServer,
		ctx:        context.Background(),
	}
	serverTran, clientTrans := mcp.NewInMemoryTransports()
	service.InitTools()
	go func() {
		if err := service.ToolServer.Run(service.ctx, serverTran); err != nil {
			fmt.Printf("server run failed: %v", err)
		}
	}()
	service.ClientSession, _ = service.Client.Connect(service.ctx, clientTrans)
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
	mcp.AddTool(server.ToolServer, mcpTool, tool.Handler())
}

func (instance *McpService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.RequestToolListEvent:
		instance.PublishToolList()
	case events.AcceptToolEvent:
		instance.ToolCall(event.Data.(types.ToolCallData))
	}
}

func (instance *McpService) ToolCall(data types.ToolCallData) {
	params := &mcp.CallToolParams{
		Name:      data.ToolName,
		Arguments: data.Paramters,
	}
	Result, err := instance.ClientSession.CallTool(instance.ctx, params)
	var builder strings.Builder
	if err != nil {
		builder.WriteString("<tool_use_error>\n")
		builder.WriteString(err.Error() + "\n")
		builder.WriteString("</tool_use_error>\n")
		PublishEvent(instance.Bus, events.ToolResultEvent, types.ToolResultData{
			RequestUUID: data.RequestUUID,
			ToolResult:  builder.String(),
		}, types.ToolService)
		return
	}
	builder.WriteString("<result>\n")
	for _, content := range Result.Content {
		builder.WriteString(content.(*mcp.TextContent).Text + "\n")
	}
	builder.WriteString("</result>\n")
	PublishEvent(instance.Bus, events.ToolResultEvent, types.ToolResultData{
		RequestUUID: data.RequestUUID,
		ToolResult:  builder.String(),
	}, types.ToolService)
}

func (instance *McpService) PublishToolList() {
	mcpToolList := make([]*mcp.Tool, 10)
	for tool := range instance.ClientSession.Tools(instance.ctx, nil) {
		mcpToolList = append(mcpToolList, tool)
	}
	PublishEvent(instance.Bus, events.UpdateToolListEvent,
		types.ToolListUpdateData{
			List: mcpToolList,
		},
		types.McpService)
}

func (instance *McpService) GetID() types.Source {
	return types.McpService
}
