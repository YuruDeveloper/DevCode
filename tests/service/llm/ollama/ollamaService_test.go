package ollama

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/service/llm/ollama"
	"DevCode/src/utils"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfig() {
	viper.Set("ollama.url", "http://localhost:11434")
	viper.Set("ollama.model", "llama3.2")
	
	tempDir := os.TempDir()
	systemPromptPath := filepath.Join(tempDir, "test_system_prompt.md")
	
	err := os.WriteFile(systemPromptPath, []byte("Test system prompt"), 0644)
	if err != nil {
		panic(err)
	}
	
	viper.Set("prompt.system", systemPromptPath)
}

func TestNewOllamaService_Success(t *testing.T) {
	setupTestConfig()
	bus, err := events.NewEventBus()
	require.NoError(t, err)

	service, err := ollama.NewOllamaService(bus)

	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, constants.LLMService, service.GetID())
}

func TestNewOllamaService_InvalidURL(t *testing.T) {
	viper.Set("ollama.url", ":/invalid-url")
	viper.Set("ollama.model", "test-model")
	
	// 임시 시스템 프롬프트 파일 생성
	tempDir := os.TempDir()
	systemPromptPath := filepath.Join(tempDir, "test_invalid_url_prompt.md")
	err := os.WriteFile(systemPromptPath, []byte("Test prompt"), 0644)
	require.NoError(t, err)
	defer os.Remove(systemPromptPath)
	
	viper.Set("prompt.system", systemPromptPath)
	
	bus, err := events.NewEventBus()
	require.NoError(t, err)

	service, err := ollama.NewOllamaService(bus)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "invalid Ollama URL")
}

func TestNewOllamaService_MissingSystemPrompt(t *testing.T) {
	viper.Set("ollama.url", "http://localhost:11434")
	viper.Set("ollama.model", "test-model")
	viper.Set("prompt.system", "")
	
	bus, err := events.NewEventBus()
	require.NoError(t, err)

	service, err := ollama.NewOllamaService(bus)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "prompt.system not configured")
}

func TestNewOllamaService_SystemPromptFileNotFound(t *testing.T) {
	viper.Set("ollama.url", "http://localhost:11434")
	viper.Set("ollama.model", "test-model")
	viper.Set("prompt.system", "/nonexistent/path/prompt.md")
	
	bus, err := events.NewEventBus()
	require.NoError(t, err)

	service, err := ollama.NewOllamaService(bus)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "fail to Read SystemPrompt")
}

func TestOllamaService_GetID(t *testing.T) {
	setupTestConfig()
	bus, err := events.NewEventBus()
	require.NoError(t, err)
	
	service, err := ollama.NewOllamaService(bus)
	require.NoError(t, err)

	id := service.GetID()

	assert.Equal(t, constants.LLMService, id)
}

// Helper function to create testable service with real managers
func createTestableService() (*TestableOllamaService, *ollama.MessageManager, *ollama.ToolManager, *ollama.StreamManager, events.Bus) {
	messageManager := ollama.NewMessageManager()
	toolManager := ollama.NewToolManager()
	streamManager := ollama.NewStreamManager()
	
	bus, _ := events.NewEventBus()
	
	service := NewTestableOllamaService(messageManager, toolManager, streamManager, bus)
	
	return service, messageManager, toolManager, streamManager, bus
}

// TestableOllamaService - 테스트 가능한 OllamaService
type TestableOllamaService struct {
	MessageManager ollama.IMessageManager
	ToolManager    ollama.IToolManager
	StreamManager  ollama.IStreamManager
	Bus            events.Bus
}

func NewTestableOllamaService(msgMgr ollama.IMessageManager, toolMgr ollama.IToolManager, streamMgr ollama.IStreamManager, bus events.Bus) *TestableOllamaService {
	return &TestableOllamaService{
		MessageManager: msgMgr,
		ToolManager:    toolMgr,
		StreamManager:  streamMgr,
		Bus:            bus,
	}
}

func (t *TestableOllamaService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.UserInputEvent:
		t.MessageManager.AddUserMessage(event.Data.(dto.UserRequestData).Message)
	case events.UpdateEnvironmentEvent:
		t.MessageManager.SetEnvironmentMessage(utils.EnvironmentUpdateDataToString(event.Data.(dto.EnvironmentUpdateData)))
	case events.UpdateToolListEvent:
		t.ToolManager.RegisterToolList(event.Data.(dto.ToolListUpdateData).List)
	case events.StreamCancelEvent:
		t.StreamManager.CancelStream(event.Data.(dto.StreamCancelData).RequestUUID)
	case events.ToolResultEvent:
		t.ProcessToolResult(event.Data.(dto.ToolResultData))
	}
}

func (t *TestableOllamaService) ProcessToolResult(data dto.ToolResultData) {
	if t.ToolManager.HasToolCall(data.RequestUUID, data.ToolCallUUID) {
		t.MessageManager.AddToolMessage(data.ToolResult)
		t.ToolManager.CompleteToolCall(data.RequestUUID, data.ToolCallUUID)
		if !t.ToolManager.HasPendingCalls(data.RequestUUID) {
			t.ToolManager.ClearRequest(data.RequestUUID)
		}
	}
}

func (t *TestableOllamaService) GetID() constants.Source {
	return constants.LLMService
}

func TestOllamaService_HandleEvent_UserInputEvent(t *testing.T) {
	service, messageManager, _, _, _ := createTestableService()
	
	requestUUID := uuid.New()
	userMessage := "안녕하세요, 테스트 메시지입니다."
	
	event := events.Event{
		Type: events.UserInputEvent,
		Data: dto.UserRequestData{
			SessionUUID: uuid.New(),
			Message:     userMessage,
			RequestUUID: requestUUID,
		},
	}

	service.HandleEvent(event)

	// 실제 매니저의 메시지 확인
	messages := messageManager.GetMessages()
	
	// 사용자 메시지가 추가되었는지 확인
	var foundUserMessage bool
	for _, msg := range messages {
		if msg.Role == "User" && msg.Content == userMessage {
			foundUserMessage = true
			break
		}
	}
	assert.True(t, foundUserMessage, "사용자 메시지가 추가되지 않았습니다")
}

func TestOllamaService_HandleEvent_UpdateEnvironmentEvent(t *testing.T) {
	service, messageManager, _, _, _ := createTestableService()
	
	environmentData := dto.EnvironmentUpdateData{
		CreateUUID:         uuid.New(),
		Cwd:                "/home/test/project",
		OS:                 "linux",
		OSVersion:          "Ubuntu 22.04",
		IsDirectoryGitRepo: true,
		TodayDate:          "2024-01-15",
	}
	
	event := events.Event{
		Type: events.UpdateEnvironmentEvent,
		Data: environmentData,
	}

	service.HandleEvent(event)

	messages := messageManager.GetMessages()
	
	// 환경 메시지가 추가되었는지 확인
	var foundEnvMessage bool
	for _, msg := range messages {
		if msg.Role == "system" && msg.Content != "" {
			if len(msg.Content) > len("Here is useful information about the environment") {
				foundEnvMessage = true
				assert.Contains(t, msg.Content, "/home/test/project")
				break
			}
		}
	}
	assert.True(t, foundEnvMessage, "환경 메시지가 추가되지 않았습니다")
}

func TestOllamaService_HandleEvent_UpdateToolListEvent(t *testing.T) {
	service, _, toolManager, _, _ := createTestableService()
	
	tools := []*mcp.Tool{
		{
			Name:        "test_tool",
			Description: "테스트용 도구",
		},
		{
			Name:        "another_tool", 
			Description: "또 다른 테스트 도구",
		},
	}
	
	event := events.Event{
		Type: events.UpdateToolListEvent,
		Data: dto.ToolListUpdateData{
			List: tools,
		},
	}

	service.HandleEvent(event)

	toolList := toolManager.GetToolList()
	assert.Equal(t, 2, len(toolList))
	assert.Equal(t, "test_tool", toolList[0].Function.Name)
	assert.Equal(t, "another_tool", toolList[1].Function.Name)
}

func TestOllamaService_HandleEvent_StreamCancelEvent(t *testing.T) {
	service, _, _, _, _ := createTestableService()
	
	requestUUID := uuid.New()
	
	event := events.Event{
		Type: events.StreamCancelEvent,
		Data: dto.StreamCancelData{
			RequestUUID: requestUUID,
		},
	}

	// StreamCancel 이벤트가 정상적으로 처리되는지 확인
	// 에러가 발생하지 않으면 성공으로 간주
	assert.NotPanics(t, func() {
		service.HandleEvent(event)
	})
}

func TestOllamaService_ProcessToolResult_ValidToolCall(t *testing.T) {
	service, messageManager, toolManager, _, _ := createTestableService()
	
	requestUUID := uuid.New()
	toolCallUUID := uuid.New()
	toolResult := "도구 실행 결과: 성공적으로 완료되었습니다."
	
	// 도구 호출 등록
	toolManager.RegisterToolCall(requestUUID, toolCallUUID, "test_tool")
	
	toolResultData := dto.ToolResultData{
		RequestUUID:  requestUUID,
		ToolCallUUID: toolCallUUID,
		ToolResult:   toolResult,
	}

	service.ProcessToolResult(toolResultData)

	// 메시지에 도구 결과가 추가되었는지 확인
	messages := messageManager.GetMessages()
	var foundToolMessage bool
	for _, msg := range messages {
		if msg.Role == "tool" && msg.Content == toolResult {
			foundToolMessage = true
			break
		}
	}
	assert.True(t, foundToolMessage, "도구 메시지가 추가되지 않았습니다")
	
	// 도구 호출이 완료되었는지 확인
	assert.False(t, toolManager.HasPendingCalls(requestUUID))
}

func TestOllamaService_ProcessToolResult_InvalidToolCall(t *testing.T) {
	service, messageManager, _, _, _ := createTestableService()
	
	requestUUID := uuid.New()
	toolCallUUID := uuid.New()
	toolResult := "유효하지 않은 도구 호출 결과"
	
	// 도구 호출을 등록하지 않음
	
	toolResultData := dto.ToolResultData{
		RequestUUID:  requestUUID,
		ToolCallUUID: toolCallUUID,
		ToolResult:   toolResult,
	}

	service.ProcessToolResult(toolResultData)

	// 유효하지 않은 도구 호출이므로 도구 메시지가 추가되지 않아야 함
	messages := messageManager.GetMessages()
	for _, msg := range messages {
		assert.NotEqual(t, "tool", msg.Role, "유효하지 않은 도구 호출에 대해 메시지가 추가되었습니다")
	}
}


// 통합 테스트
func TestOllamaService_CompleteWorkflow(t *testing.T) {
	service, messageManager, toolManager, _, bus := createTestableService()
	
	// 이벤트 구독 설정
	bus.Subscribe(events.UpdateEnvironmentEvent, service)
	bus.Subscribe(events.UpdateToolListEvent, service)
	bus.Subscribe(events.UserInputEvent, service)
	bus.Subscribe(events.ToolResultEvent, service)
	
	// 1. 환경 업데이트 이벤트
	environmentData := dto.EnvironmentUpdateData{
		CreateUUID:         uuid.New(),
		Cwd:                "/workspace/test",
		OS:                 "linux",
		OSVersion:          "Ubuntu 22.04",
		IsDirectoryGitRepo: true,
		TodayDate:          "2024-01-15",
	}
	
	service.HandleEvent(events.Event{
		Type: events.UpdateEnvironmentEvent,
		Data: environmentData,
	})
	
	// 2. 도구 리스트 업데이트 이벤트
	tools := []*mcp.Tool{
		{Name: "file_reader", Description: "파일 읽기 도구"},
		{Name: "calculator", Description: "계산 도구"},
	}
	
	service.HandleEvent(events.Event{
		Type: events.UpdateToolListEvent,
		Data: dto.ToolListUpdateData{
			List: tools,
		},
	})
	
	// 3. 사용자 입력 이벤트
	requestUUID := uuid.New()
	userMessage := "안녕하세요! 파일을 읽어주세요."
	service.HandleEvent(events.Event{
		Type: events.UserInputEvent,
		Data: dto.UserRequestData{
			SessionUUID: uuid.New(),
			Message:     userMessage,
			RequestUUID: requestUUID,
		},
	})
	
	// 4. 도구 결과 이벤트
	toolCallUUID := uuid.New()
	toolResult := "파일 내용: Hello, World!"
	
	toolManager.RegisterToolCall(requestUUID, toolCallUUID, "file_reader")
	
	service.HandleEvent(events.Event{
		Type: events.ToolResultEvent,
		Data: dto.ToolResultData{
			RequestUUID:  requestUUID,
			ToolCallUUID: toolCallUUID,
			ToolResult:   toolResult,
		},
	})
	
	// 결과 검증
	messages := messageManager.GetMessages()
	toolList := toolManager.GetToolList()
	
	// 도구 리스트 확인
	assert.Equal(t, 2, len(toolList))
	
	// 메시지 확인
	var hasEnvMessage, hasUserMessage, hasToolMessage bool
	for _, msg := range messages {
		if msg.Role == "system" && len(msg.Content) > 50 {
			hasEnvMessage = true
		}
		if msg.Role == "User" && msg.Content == userMessage {
			hasUserMessage = true
		}
		if msg.Role == "tool" && msg.Content == toolResult {
			hasToolMessage = true
		}
	}
	
	assert.True(t, hasEnvMessage, "환경 메시지가 없습니다")
	assert.True(t, hasUserMessage, "사용자 메시지가 없습니다")
	assert.True(t, hasToolMessage, "도구 메시지가 없습니다")
	assert.False(t, toolManager.HasPendingCalls(requestUUID), "도구 호출이 완료되지 않았습니다")
}