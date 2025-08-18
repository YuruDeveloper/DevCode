package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/types"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
)

// assertOllamaServiceIsValid validates OllamaService initialization
func assertOllamaServiceIsValid(t *testing.T, ollamaService *service.OllamaService, bus *events.EventBus) {
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
}

// assertEnvironmentMessageIsValid validates environment message content
func assertEnvironmentMessageIsValid(t *testing.T, message *api.Message, environment string) {
	if message == nil {
		t.Fatal("EnviromentMessage가 nil을 반환했습니다")
	}

	if message.Role != "system" {
		t.Error("메시지 역할이 'system'이 아닙니다")
	}

	expectedContent := service.EnviromentInfo + environment
	if message.Content != expectedContent {
		t.Error("메시지 내용이 올바르지 않습니다")
	}
}

// assertUserMessageAdded validates that user message was added correctly
func assertUserMessageAdded(t *testing.T, ollamaService *service.OllamaService, expectedMessage string) {
	if len(ollamaService.Messages) == 0 {
		t.Fatal("메시지가 추가되지 않았습니다")
	}

	lastMessage := ollamaService.Messages[len(ollamaService.Messages)-1]
	if lastMessage.Role != "user" {
		t.Error("메시지 역할이 'user'가 아닙니다")
	}

	if lastMessage.Content != expectedMessage {
		t.Error("메시지 내용이 올바르지 않습니다")
	}
}



func TestNewOllamaService_ShouldCreateServiceSuccessfully(t *testing.T) {
	// Given & When
	ollamaService, eventBus := SetupOllamaService()
	defer CleanupTestConfig()

	// Then
	AssertNoPanic(t, "NewOllamaService", func() {
		service.NewOllamaService(eventBus)
	})

	assertOllamaServiceIsValid(t, ollamaService, eventBus)
}

func TestOllamaService_GetID_ShouldReturnCorrectServiceID(t *testing.T) {
	// Given
	ollamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()
	expectedID := types.LLMService

	// When
	actualID := ollamaService.GetID()

	// Then
	if int(actualID) != int(expectedID) {
		t.Errorf("GetID() = %d, 예상값 %d", int(actualID), int(expectedID))
	}
}

func TestOllamaService_EnvironmentMessage_ShouldCreateSystemMessage(t *testing.T) {
	// Given
	ollamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()
	testEnvironment := "test environment info"
	ollamaService.Environment = testEnvironment

	// When
	message := ollamaService.EnviromentMessage()

	// Then
	assertEnvironmentMessageIsValid(t, message, testEnvironment)
}

func TestOllamaService_UpdateUserInput_ShouldAddUserMessage(t *testing.T) {
	// Given
	ollamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()
	testMessage := "테스트 사용자 입력"

	// When
	ollamaService.UpdateUserInput(testMessage)

	// Then
	assertUserMessageAdded(t, ollamaService, testMessage)
}


func TestOllamaService_UpdateEnvironmentToolList_ShouldPublishBothEvents(t *testing.T) {
	// Given
	ollamaService, eventBus := SetupOllamaService()
	defer CleanupTestConfig()

	var receivedEvents []events.Event
	testHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			receivedEvents = append(receivedEvents, event)
		},
		ID: TestService,
	}

	eventBus.Subscribe(events.RequestEnvionmentvent, testHandler)
	eventBus.Subscribe(events.RequestToolListEvent, testHandler)

	// When
	ollamaService.UpdateEnviromentToolList()
	time.Sleep(AsyncWaitTime)

	// Then
	assertBothEventsWerePublished(t, receivedEvents)
}

// assertBothEventsWerePublished validates that both environment and tool list events were published
func assertBothEventsWerePublished(t *testing.T, eventList []events.Event) {
	if len(eventList) != 2 {
		t.Errorf("발행된 이벤트 수 = %d, 예상값 2", len(eventList))
	}

	hasEnvironmentEvent := false
	hasToolListEvent := false
	for _, event := range eventList {
		if event.Type == events.RequestEnvionmentvent {
			hasEnvironmentEvent = true
		}
		if event.Type == events.RequestToolListEvent {
			hasToolListEvent = true
		}
	}

	if !hasEnvironmentEvent {
		t.Error("RequestEnvionmentvent가 발행되지 않았습니다")
	}

	if !hasToolListEvent {
		t.Error("RequestToolListEvent가 발행되지 않았습니다")
	}
}


func TestOllamaService_CancelStream_ShouldCancelActiveStream(t *testing.T) {
	// Given
	ollamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()
	
	requestUUID := uuid.New()
	ctx, cancelFunc := context.WithCancel(context.Background())
	
	ollamaService.ActiveStreams = make(map[uuid.UUID]context.CancelFunc)
	ollamaService.ActiveStreams[requestUUID] = cancelFunc

	// When
	ollamaService.CancelStream(requestUUID)

	// Then
	select {
	case <-ctx.Done():
		// Success: context was cancelled
	default:
		t.Error("context가 취소되지 않았습니다")
	}
}

func TestOllamaService_HandleEvent_ShouldProcessUserInputEventCorrectly(t *testing.T) {
	// Given
	ollamaService, _ := SetupOllamaService()
	defer CleanupTestConfig()

	testMessage := "테스트 메시지"
	requestData := CreateTestRequestData(testMessage)

	userInputEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      requestData,
		Timestamp: time.Now(),
		Source:    TestService,
	}

	// When & Then
	AssertNoPanic(t, "HandleEvent", func() {
		ollamaService.HandleEvent(userInputEvent)
	})

	assertUserMessageAddedFromEvent(t, ollamaService, testMessage)
}

// assertUserMessageAddedFromEvent validates that user message from event was processed correctly
func assertUserMessageAddedFromEvent(t *testing.T, ollamaService *service.OllamaService, expectedMessage string) {
	if len(ollamaService.Messages) == 0 {
		t.Error("메시지가 추가되지 않았습니다")
		return
	}

	lastMessage := ollamaService.Messages[len(ollamaService.Messages)-1]
	if lastMessage.Content != expectedMessage {
		t.Error("메시지 내용이 올바르지 않습니다")
	}
}

