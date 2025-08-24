package mcp

import (
	"UniCode/src/constants"
	"UniCode/src/dto"
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/tools/read"
	"UniCode/src/types"
	"context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type McpService struct {
	client        *mcp.Client
	clientSession *mcp.ClientSession
	toolServer    *mcp.Server
	bus           *events.EventBus
	ctx           context.Context
	logger        *zap.Logger
}

func NewMcpService(bus *events.EventBus, logger *zap.Logger) *McpService {
	logger.Info("McpService 초기화 시작")

	requireds := []string{"mcp.name", "mcp.version", "server.name", "server.version"}
	data := make([]string,4)
	for index, required := range requireds {
		data[index] = viper.GetString(required)
	}

	logger.Debug("MCP 설정 로드 완료",
		zap.String("mcp.name", data[0]),
		zap.String("mcp.version", data[1]),
		zap.String("server.name", data[2]),
		zap.String("server.version", data[3]))

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
		logger:     logger,
	}

	serverTran, clientTrans := mcp.NewInMemoryTransports()

	service.InitTools()

	go func() {
		logger.Info("MCP 툴 서버 시작")
		if err := service.toolServer.Run(service.ctx, serverTran); err != nil {
			logger.Error("server run failed", zap.Error(err))
		} else {
			logger.Info("MCP 툴 서버 실행 완료")
		}
	}()

	logger.Info("MCP 클라이언트 연결 시도")
	service.clientSession, _ = service.client.Connect(service.ctx, clientTrans)
	logger.Info("MCP 클라이언트 연결 완료")

	bus.Subscribe(events.RequestToolListEvent, service)
	bus.Subscribe(events.AcceptToolEvent, service)
	logger.Info("이벤트 구독 완료")

	logger.Info("McpService 초기화 완료")
	return service
}

func (instance *McpService) InitTools() {
	instance.logger.Info("도구 초기화 시작")
	InsertTool(instance, &read.Tool{})
	instance.logger.Info("도구 초기화 완료")
}

func InsertTool[T any](server *McpService, tool types.Tool[T]) {
	server.logger.Debug("도구 등록 중", zap.String("tool_name", tool.Name()))
	mcpTool := &mcp.Tool{
		Name:        tool.Name(),
		Description: tool.Description(),
	}
	mcp.AddTool(server.toolServer, mcpTool, tool.Handler())
	server.logger.Info("도구 등록 완료",
		zap.String("tool_name", tool.Name()),
		zap.String("description", tool.Description()))
}

func (instance *McpService) HandleEvent(event events.Event) {
	instance.logger.Debug("이벤트 수신", zap.String("event_type", event.Type.String()))

	switch event.Type {
	case events.RequestToolListEvent:
		instance.logger.Info("도구 목록 요청 처리 중")
		instance.PublishToolList()
	case events.AcceptToolEvent:
		instance.logger.Info("도구 호출 요청 처리 중")
		instance.ToolCall(event.Data.(dto.ToolCallData))
	default:
		instance.logger.Warn("알 수 없는 이벤트 타입", zap.String("event_type", event.Type.String()))
	}
}

func (instance *McpService) ToolCall(data dto.ToolCallData) {
	instance.logger.Info("도구 호출 시작",
		zap.String("tool_name", data.ToolName),
		zap.String("request_uuid", data.RequestUUID.String()))

	params := &mcp.CallToolParams{
		Name:      data.ToolName,
		Arguments: data.Parameters,
	}

	result, err := instance.clientSession.CallTool(instance.ctx, params)

	if err != nil {
		instance.logger.Error("도구 호출 실패",
			zap.String("tool_name", data.ToolName),
			zap.String("request_uuid", data.RequestUUID.String()),
			zap.Error(err))
	} else {
		instance.logger.Info("도구 호출 성공",
			zap.String("tool_name", data.ToolName),
			zap.String("request_uuid", data.RequestUUID.String()))
	}

	service.PublishEvent(instance.bus, events.ToolRawResultEvent, dto.ToolRawResultData{
		RequestUUID: data.RequestUUID,
		ToolCall:    data.ToolCallUUID,
		Result:      result,
	}, constants.McpService)
}

func (instance *McpService) PublishToolList() {
	instance.logger.Info("도구 목록 발행 시작")

	mcpToolList := make([]*mcp.Tool, 0, 10)
	for tool := range instance.clientSession.Tools(instance.ctx, nil) {
		mcpToolList = append(mcpToolList, tool)
		instance.logger.Debug("도구 발견", zap.String("tool_name", tool.Name))
	}

	instance.logger.Info("도구 목록 발행 완료", zap.Int("tool_count", len(mcpToolList)))

	service.PublishEvent(instance.bus, events.UpdateToolListEvent,
		dto.ToolListUpdateData{
			List: mcpToolList,
		},
		constants.McpService)
}

func (instance *McpService) GetID() constants.Source {
	return constants.McpService
}
