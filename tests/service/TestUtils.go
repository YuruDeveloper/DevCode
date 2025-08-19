package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/types"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
	"github.com/spf13/viper"
)

// Constants for testing
const (
	TestTimeout = 30 * time.Second
	AsyncWaitTime = 10 * time.Millisecond
	IntegrationTimeout = 60 * time.Second
	OllamaHealthCheckTimeout = 2 * time.Second
)

// Test file paths
const (
	TestSystemPromptPath = "/tmp/test_system_prompt.md"
	IntegrationSystemPromptPath = "/tmp/integration_test_system_prompt.md"
)

// TestService constant for testing
const TestService types.Source = 999

// TestEventHandler is a test implementation of EventHandler interface
type TestEventHandler struct {
	HandleFunc func(events.Event)
	ID         types.Source
}

func (handler *TestEventHandler) HandleEvent(event events.Event) {
	if handler.HandleFunc != nil {
		handler.HandleFunc(event)
	}
}

func (handler *TestEventHandler) GetID() types.Source {
	return handler.ID
}

// ServiceSetup provides unified setup for all services
type ServiceSetup struct {
	MessageService     *service.MessageService
	OllamaService      *service.OllamaService
	HistoryService     *service.HistoryService
	EnvironmentService *service.EnvironmentService
	EventBus           *events.EventBus
}

// SetupAllServices creates all services for comprehensive testing
func SetupAllServices() *ServiceSetup {
	eventBus := events.NewEventBus()
	
	return &ServiceSetup{
		MessageService:     service.NewMessageService(eventBus),
		OllamaService:      setupOllamaServiceInternal(eventBus),
		HistoryService:     service.NewHistoryService(eventBus),
		EnvironmentService: service.NewEnvironmentService(eventBus),
		EventBus:           eventBus,
	}
}

// SetupMessageService creates a new MessageService for testing
func SetupMessageService() (*service.MessageService, *events.EventBus) {
	eventBus := events.NewEventBus()
	messageService := service.NewMessageService(eventBus)
	return messageService, eventBus
}

// SetupOllamaService creates a new OllamaService for testing
func SetupOllamaService() (*service.OllamaService, *events.EventBus) {
	setupTestConfig()
	eventBus := events.NewEventBus()
	ollamaService := service.NewOllamaService(eventBus)
	return ollamaService, eventBus
}

// SetupHistoryService creates a new HistoryService for testing
func SetupHistoryService() (*service.HistoryService, *events.EventBus) {
	eventBus := events.NewEventBus()
	historyService := service.NewHistoryService(eventBus)
	return historyService, eventBus
}

// SetupEnvironmentService creates a new EnvironmentService for testing
func SetupEnvironmentService() (*service.EnvironmentService, *events.EventBus) {
	eventBus := events.NewEventBus()
	environmentService := service.NewEnvironmentService(eventBus)
	return environmentService, eventBus
}

// SetupToolService creates a new ToolService for testing
func SetupToolService() (*service.ToolService, *events.EventBus) {
	setupTestConfig()
	eventBus := events.NewEventBus()
	toolService := service.NewToolService(eventBus)
	return toolService, eventBus
}

// Internal setup function for OllamaService
func setupOllamaServiceInternal(eventBus *events.EventBus) *service.OllamaService {
	setupTestConfig()
	return service.NewOllamaService(eventBus)
}

// SetupTestConfig configures test environment for OllamaService tests
func setupTestConfig() {
	viper.Set("ollama.url", "localhost:11434")
	viper.Set("ollama.model", "test-model")
	viper.Set("prompt.system", TestSystemPromptPath)
	
	systemPrompt := "You are a helpful assistant for testing."
	err := os.WriteFile(TestSystemPromptPath, []byte(systemPrompt), 0644)
	if err != nil {
		panic(err)
	}
}

// CleanupTestConfig removes test configuration files
func CleanupTestConfig() {
	os.Remove(TestSystemPromptPath)
	os.Remove(IntegrationSystemPromptPath)
}

// SetupIntegrationTest configures test environment for integration tests
func SetupIntegrationTest() {
	viper.Set("ollama.url", "http://localhost:11434")
	viper.Set("ollama.model", "llama3.1:8b")
	viper.Set("prompt.system", "/home/cecil/UniCode/SystemPrompt/Root.md")
}

// IsOllamaRunning checks if Ollama server is running
func IsOllamaRunning() bool {
	timeout := time.After(OllamaHealthCheckTimeout)
	done := make(chan bool)
	
	go func() {
		resp, err := http.Get("http://localhost:11434/api/tags")
		if err == nil {
			resp.Body.Close()
			done <- true
		} else {
			done <- false
		}
	}()
	
	select {
	case result := <-done:
		return result
	case <-timeout:
		return false
	}
}

// AssertNoPanic ensures that a function does not panic during execution
func AssertNoPanic(t *testing.T, functionName string, fn func()) {
	defer func() {
		if recovery := recover(); recovery != nil {
			t.Errorf("%s에서 패닉 발생: %v", functionName, recovery)
		}
	}()
	fn()
}

// CreateTestEnvironmentData creates test environment data for testing
func CreateTestEnvironmentData() types.EnvironmentUpdateData {
	return types.EnvironmentUpdateData{
		CreateUUID:         uuid.New(),
		Cwd:                "/test/path",
		OS:                 "linux",
		OSVersion:          "5.4.0",
		IsDirectoryGitRepo: true,
		TodayDate:          "2024-01-01",
	}
}

// CreateTestRequestData creates test request data for testing
func CreateTestRequestData(message string) types.RequestData {
	return types.RequestData{
		SessionUUID: uuid.New(),
		RequestUUID: uuid.New(),
		Message:     message,
	}
}

// CreateTestStreamStartEvent creates a test stream start event
func CreateTestStreamStartEvent() events.Event {
	return events.Event{
		Type:      events.StreamStartEvent,
		Data:      types.StreamStartData{RequestUUID: uuid.New()},
		Timestamp: time.Now(),
		Source:    TestService,
	}
}

// CreateTestToolCallEvent creates a test tool call event
func CreateTestToolCallEvent() events.Event {
	return events.Event{
		Type: events.ToolCallEvent,
		Data: types.ToolCallData{
			RequestUUID: uuid.New(),
			ToolName:    "testTool",
			Parameters:   map[string]any{"param1": "value1", "param2": 42},
		},
		Timestamp: time.Now(),
		Source:    TestService,
	}
}

// WaitForEventCompletion waits for specific event completion with timeout
func WaitForEventCompletion(t *testing.T, eventList *[]events.Event, eventType events.EventType, timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	
	for {
		select {
		case <-timeoutChan:
			t.Errorf("이벤트 대기 시간 초과")
			return false
		case <-time.After(AsyncWaitTime):
			for _, event := range *eventList {
				if event.Type == eventType {
					return true
				}
				if event.Type == events.StreamErrorEvent {
					t.Errorf("스트림 에러 발생: %v", event.Data)
					return false
				}
			}
		}
	}
}

// CountEventsByType counts events by their type
func CountEventsByType(eventList []events.Event, eventType events.EventType) int {
	count := 0
	for _, event := range eventList {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CreateTestTool creates a test tool for getCurrentTime
func CreateTestTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "getCurrentTime",
			Description: "현재 시간을 반환합니다",
			Parameters: struct {
				Type       string                     `json:"type"`
				Defs       any                        `json:"$defs,omitempty"`
				Items      any                        `json:"items,omitempty"`
				Required   []string                   `json:"required"`
				Properties map[string]api.ToolProperty `json:"properties"`
			}{
				Type: "object",
				Properties: map[string]api.ToolProperty{
					"format": {
						Type:        []string{"string"},
						Description: "시간 형식 (예: 'YYYY-MM-DD HH:mm:ss')",
					},
				},
			},
		},
	}
}

// ToolCallTestHandler handles tool call and stream events for testing
type ToolCallTestHandler struct {
	ToolCallEvents []events.Event
	StreamEvents   []events.Event
	T              *testing.T
}

func NewToolCallTestHandler(t *testing.T) *ToolCallTestHandler {
	return &ToolCallTestHandler{
		ToolCallEvents: make([]events.Event, 0),
		StreamEvents:   make([]events.Event, 0),
		T:              t,
	}
}

func (h *ToolCallTestHandler) HandleEvent(event events.Event) {
	switch event.Type {
	case events.ToolCallEvent:
		h.ToolCallEvents = append(h.ToolCallEvents, event)
	case events.StreamCompleteEvent:
		h.StreamEvents = append(h.StreamEvents, event)
		h.logToolCallsInStreamEvent(event)
	}
}

func (h *ToolCallTestHandler) GetID() types.Source {
	return TestService
}

func (h *ToolCallTestHandler) logToolCallsInStreamEvent(event events.Event) {
	if data, ok := event.Data.(types.StreamCompleteData); ok {
		if len(data.FinalMessage.ToolCalls) > 0 {
			h.T.Logf("응답에 도구 호출이 포함되어 있습니다: %d개", len(data.FinalMessage.ToolCalls))
			for i, call := range data.FinalMessage.ToolCalls {
				h.T.Logf("도구 호출 %d: %s, 매개변수: %+v", i+1, call.Function.Name, call.Function.Arguments)
			}
		}
	}
}

// WaitForToolCallCompletion waits for tool call integration test completion
func WaitForToolCallCompletion(t *testing.T, handler *ToolCallTestHandler, timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	
	for {
		select {
		case <-timeoutChan:
			t.Error("API 응답 시간 초과")
			return false
		case <-time.After(200 * time.Millisecond):
			if len(handler.StreamEvents) > 0 {
				return true
			}
		}
	}
}

// ValidateToolCallResults validates tool call test results
func ValidateToolCallResults(t *testing.T, handler *ToolCallTestHandler) {
	t.Logf("API 호출이 성공적으로 완료되었습니다")
	
	// 도구 호출 이벤트 검증
	if len(handler.ToolCallEvents) > 0 {
		t.Logf("✅ ToolCallEvent가 %d개 발생했습니다!", len(handler.ToolCallEvents))
		for i, event := range handler.ToolCallEvents {
			if data, ok := event.Data.(types.ToolCallData); ok {
				t.Logf("   도구 호출 %d: %s, 매개변수: %+v", i+1, data.ToolName, data.Parameters)
			}
		}
	} else {
		t.Logf("❌ ToolCallEvent가 발생하지 않았습니다")
	}
	
	// StreamCompleteEvent에서 최종 메시지 확인
	for _, event := range handler.StreamEvents {
		if data, ok := event.Data.(types.StreamCompleteData); ok {
			t.Logf("최종 응답 내용: %s", data.FinalMessage.Content)
			if len(data.FinalMessage.ToolCalls) > 0 {
				t.Logf("✅ 최종 메시지에 도구 호출이 포함되어 있습니다: %d개", len(data.FinalMessage.ToolCalls))
			} else {
				t.Logf("❌ 최종 메시지에 도구 호출이 포함되지 않았습니다")
			}
		}
	}
}