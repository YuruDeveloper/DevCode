package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/types"
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
	"github.com/spf13/viper"
)


// SetupTestConfig configures test environment for OllamaService tests
func SetupTestConfig() {
	viper.Set("ollama.url", "localhost:11434")
	viper.Set("ollama.model", "test-model")
	viper.Set("prompt.system", "/tmp/test_system_prompt.md")
	
	SystemPrompt := "You are a helpful assistant for testing."
	err := os.WriteFile("/tmp/test_system_prompt.md", []byte(SystemPrompt), 0644)
	if err != nil {
		panic(err)
	}
}

// CleanupTestConfig removes test configuration files
func CleanupTestConfig() {
	os.Remove("/tmp/test_system_prompt.md")
}

// SetupOllamaService creates a new OllamaService for testing
func SetupOllamaService() (*service.OllamaService, *events.EventBus) {
	SetupTestConfig()
	EventBus := events.NewEventBus()
	OllamaService := service.NewOllamaService(EventBus)
	return OllamaService, EventBus
}



func TestNewOllamaService_ShouldCreateServiceSuccessfully(t *testing.T) {
	// Given & When
	OllamaService, EventBus := SetupOllamaService()
	defer CleanupTestConfig()

	// Then
	AssertNoPanic(t, "NewOllamaService", func() {
		service.NewOllamaService(EventBus)
	})

	AssertOllamaServiceIsValid(t, OllamaService, EventBus)
}

func TestOllamaService_GetID_ShouldReturnCorrectServiceID(t *testing.T) {
	// Given
	OllamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()
	ExpectedID := types.LLMService

	// When
	ActualID := OllamaService.GetID()

	// Then
	if int(ActualID) != int(ExpectedID) {
		t.Errorf("GetID() = %d, 예상값 %d", int(ActualID), int(ExpectedID))
	}
}

func TestOllamaService_EnvironmentMessage_ShouldCreateSystemMessage(t *testing.T) {
	// Given
	OllamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()
	OllamaService.Environment = "test environment info"

	// When
	Message := OllamaService.EnviromentMessage()

	// Then
	AssertEnvironmentMessageIsValid(t, Message, OllamaService.Environment)
}

func TestOllamaService_UpdateUserInput_ShouldAddUserMessage(t *testing.T) {
	// Given
	OllamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()
	TestMessage := "테스트 사용자 입력"

	// When
	OllamaService.UpdateUserInput(TestMessage)

	// Then
	AssertUserMessageAdded(t, OllamaService, TestMessage)
}

// AssertOllamaServiceIsValid validates OllamaService initialization
func AssertOllamaServiceIsValid(t *testing.T, ollamaService *service.OllamaService, bus *events.EventBus) {
	if ollamaService == nil {
		t.Fatal("NewOllamaService는 nil을 반환하면 안됩니다")
	}

	if ollamaService.Bus != bus {
		t.Error("OllamaService의 Bus가 올바르게 설정되지 않았습니다")
	}

	if ollamaService.Model != "test-model" {
		t.Error("Model이 올바르게 설정되지 않았습니다")
	}

	if ollamaService.Client == nil {
		t.Error("Client가 설정되지 않았습니다")
	}

	if ollamaService.Ctx == nil {
		t.Error("Context가 설정되지 않았습니다")
	}

	if ollamaService.SystemPrompt == "" {
		t.Error("SystemPrompt가 설정되지 않았습니다")
	}
}

// AssertEnvironmentMessageIsValid validates environment message content
func AssertEnvironmentMessageIsValid(t *testing.T, message *api.Message, environment string) {
	if message == nil {
		t.Fatal("EnviromentMessage가 nil을 반환했습니다")
	}

	if message.Role != "system" {
		t.Error("메시지 역할이 'system'이 아닙니다")
	}

	ExpectedContent := service.EnviromentInfo + environment
	if message.Content != ExpectedContent {
		t.Error("메시지 내용이 올바르지 않습니다")
	}
}

// AssertUserMessageAdded validates that user message was added correctly
func AssertUserMessageAdded(t *testing.T, ollamaService *service.OllamaService, expectedMessage string) {
	if len(ollamaService.Messages) == 0 {
		t.Fatal("메시지가 추가되지 않았습니다")
	}

	LastMessage := ollamaService.Messages[len(ollamaService.Messages)-1]
	if LastMessage.Role != "user" {
		t.Error("메시지 역할이 'user'가 아닙니다")
	}

	if LastMessage.Content != expectedMessage {
		t.Error("메시지 내용이 올바르지 않습니다")
	}
}

func TestOllamaService_UpdateEnvironmentToolList_ShouldPublishBothEvents(t *testing.T) {
	// Given
	OllamaService, EventBus := SetupOllamaService()
	defer CleanupTestConfig()

	var ReceivedEvents []events.Event
	TestHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			ReceivedEvents = append(ReceivedEvents, event)
		},
		ID: TestService,
	}

	EventBus.Subscribe(events.RequestEnvionmentvent, TestHandler)
	EventBus.Subscribe(events.RequestToolListEvent, TestHandler)

	// When
	OllamaService.UpdateEnviromentToolList()
	time.Sleep(10 * time.Millisecond) // Wait for async processing

	// Then
	AssertBothEventsWerePublished(t, ReceivedEvents)
}

// AssertBothEventsWerePublished validates that both environment and tool list events were published
func AssertBothEventsWerePublished(t *testing.T, eventList []events.Event) {
	if len(eventList) != 2 {
		t.Errorf("발행된 이벤트 수 = %d, 예상값 2", len(eventList))
	}

	HasEnvironmentEvent := false
	HasToolListEvent := false
	for _, event := range eventList {
		if event.Type == events.RequestEnvionmentvent {
			HasEnvironmentEvent = true
		}
		if event.Type == events.RequestToolListEvent {
			HasToolListEvent = true
		}
	}

	if !HasEnvironmentEvent {
		t.Error("RequestEnvionmentvent가 발행되지 않았습니다")
	}

	if !HasToolListEvent {
		t.Error("RequestToolListEvent가 발행되지 않았습니다")
	}
}

func TestOllamaService_CancelStream_ShouldCancelActiveStream(t *testing.T) {
	// Given
	OllamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()
	
	RequestUUID := uuid.New()
	Context, CancelFunc := context.WithCancel(context.Background())
	
	OllamaService.ActiveStreams = make(map[uuid.UUID]context.CancelFunc)
	OllamaService.ActiveStreams[RequestUUID] = CancelFunc

	// When
	OllamaService.CancelStream(RequestUUID)

	// Then
	select {
	case <-Context.Done():
		// Success: context was cancelled
	default:
		t.Error("context가 취소되지 않았습니다")
	}
}

func TestOllamaService_HandleEvent_ShouldProcessUserInputEventCorrectly(t *testing.T) {
	// Given
	OllamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()

	RequestData := types.RequestData{
		SessionUUID: uuid.New(),
		RequestUUID: uuid.New(),
		Message:     "테스트 메시지",
	}

	UserInputEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      RequestData,
		Timestamp: time.Now(),
		Source:    TestService,
	}

	// When & Then
	AssertNoPanic(t, "HandleEvent", func() {
		OllamaService.HandleEvent(UserInputEvent)
	})

	AssertUserMessageAddedFromEvent(t, OllamaService, RequestData.Message)
}

// AssertUserMessageAddedFromEvent validates that user message from event was processed correctly
func AssertUserMessageAddedFromEvent(t *testing.T, ollamaService *service.OllamaService, expectedMessage string) {
	if len(ollamaService.Messages) == 0 {
		t.Error("메시지가 추가되지 않았습니다")
		return
	}

	LastMessage := ollamaService.Messages[len(ollamaService.Messages)-1]
	if LastMessage.Content != expectedMessage {
		t.Error("메시지 내용이 올바르지 않습니다")
	}
}