package service

import (
	"UniCode/src/events"
	"UniCode/src/tools/read"
	"UniCode/src/types"
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
)

type McpService struct {
	Client *mcp.Client
	ClientSession *mcp.ClientSession
	ToolServer *mcp.Server
	Bus *events.EventBus
	ctx context.Context
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

	Service := &McpService{
		Client: mcpClient,
		Bus: bus,
		ToolServer: mcpServer,
		ctx: context.Background(),
	}
	Service.InitTools()
	Service.ClientSession , _ = Service.Client.Connect(Service.ctx,&mcp.CommandTransport{})
	bus.Subscribe(events.RequestToolListEvent,Service)
	return Service
}

func (instance *McpService) InitTools() {
	InsertTool(instance,&read.Tool{})
	instance.ToolServer.Run(context.Background(),&mcp.InMemoryTransport{})
}

func  InsertTool[T any](server *McpService,tool types.Tool[T]) {
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
	}
}

func (instance *McpService) PublishToolList() {
	mcpToolList := make([]*mcp.Tool,10)
	for tool := range instance.ClientSession.Tools(instance.ctx,nil) {
		mcpToolList = append(mcpToolList, tool)
	}
	instance.Bus.Publish(
		events.Event{
			Type: events.UpdateToolListEvent,
			Timestamp: time.Now(),
			Data: types.ToolListUpdateData {
				List: mcpToolList,
			},
			Source: types.McpService,
		},
	)
}

func (instance *McpService) GetID() types.Source {
	return types.McpService
}