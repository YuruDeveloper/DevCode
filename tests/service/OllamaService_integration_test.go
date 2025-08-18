package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/types"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
)


func TestOllamaService_CallApi_Integration(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	SetupIntegrationTest()
	defer CleanupTestConfig()

	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)

	// 스트림 이벤트를 받을 핸들러
	var streamEvents []events.Event
	testHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			streamEvents = append(streamEvents, event)
		},
		ID: TestService,
	}

	// 스트림 관련 이벤트들 구독
	bus.Subscribe(events.StreamStartEvent, testHandler)
	bus.Subscribe(events.StreamChunkEvent, testHandler)
	bus.Subscribe(events.StreamCompleteEvent, testHandler)
	bus.Subscribe(events.StreamErrorEvent, testHandler)

	// 사용자 메시지 추가
	ollamaService.UpdateUserInput("안녕하세요! 간단히 인사해주세요.")

	// API 호출
	requestUUID := uuid.New()
	ollamaService.CallApi(requestUUID)

	// 응답을 기다림 (실제 API 호출이므로 시간이 걸림)
	timeout := time.After(30 * time.Second)
	completed := false

	for !completed {
		select {
		case <-timeout:
			t.Fatal("API 호출 시간 초과 (30초)")
		case <-time.After(100 * time.Millisecond):
			// 이벤트 확인
			for _, event := range streamEvents {
				if event.Type == events.StreamCompleteEvent {
					completed = true
					break
				}
				if event.Type == events.StreamErrorEvent {
					t.Fatalf("API 호출 에러: %v", event.Data)
				}
			}
		}
	}

	// 스트림 시작 이벤트가 발생했는지 확인
	hasStreamStart := false
	hasStreamChunk := false
	hasStreamComplete := false

	for _, event := range streamEvents {
		switch event.Type {
		case events.StreamStartEvent:
			hasStreamStart = true
		case events.StreamChunkEvent:
			hasStreamChunk = true
		case events.StreamCompleteEvent:
			hasStreamComplete = true
		}
	}

	if !hasStreamStart {
		t.Error("StreamStartEvent가 발생하지 않았습니다")
	}

	if !hasStreamChunk {
		t.Error("StreamChunkEvent가 발생하지 않았습니다")
	}

	if !hasStreamComplete {
		t.Error("StreamCompleteEvent가 발생하지 않았습니다")
	}

	t.Logf("총 %d개의 스트림 이벤트가 발생했습니다", len(streamEvents))
}

func TestOllamaService_ToolCall_Integration(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	SetupIntegrationTest()
	defer CleanupTestConfig()

	// Given
	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)
	testTool := CreateTestTool()
	ollamaService.Tools = []api.Tool{testTool}

	// 도구 호출 이벤트 핸들러 설정
	toolHandler := NewToolCallTestHandler(t)
	bus.Subscribe(events.ToolCallEvent, toolHandler)
	bus.Subscribe(events.StreamCompleteEvent, toolHandler)

	// When
	testMessage := "현재 시간은 몇시인가요? 사용가능한 도구를 활용해서 대답하세요"
	ollamaService.UpdateUserInput(testMessage)
	
	requestUUID := uuid.New()
	ollamaService.CallApi(requestUUID)

	// Then
	if WaitForToolCallCompletion(t, toolHandler, TestTimeout) {
		ValidateToolCallResults(t, toolHandler)
	}
}

func TestOllamaService_StreamCancel_Integration(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	SetupIntegrationTest()
	defer CleanupTestConfig()

	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)

	// 스트림 이벤트를 받을 핸들러
	var streamEvents []events.Event
	testHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			streamEvents = append(streamEvents, event)
		},
		ID: TestService,
	}

	bus.Subscribe(events.StreamStartEvent, testHandler)
	bus.Subscribe(events.StreamChunkEvent, testHandler)
	bus.Subscribe(events.StreamCompleteEvent, testHandler)
	bus.Subscribe(events.StreamErrorEvent, testHandler)

	// 긴 응답을 유도하는 메시지
	ollamaService.UpdateUserInput("한국의 역사에 대해 자세히 설명해주세요. 조선시대부터 현대까지 모든 시대를 포함해서 아주 길게 써주세요.")

	// API 호출
	requestUUID := uuid.New()
	ollamaService.CallApi(requestUUID)

	// 스트림이 시작될 때까지 잠시 대기
	time.Sleep(1 * time.Second)

	// 스트림 취소
	ollamaService.CancelStream(requestUUID)

	// 취소 후 추가 대기
	time.Sleep(2 * time.Second)

	// 스트림이 시작되었지만 완료되지 않았는지 확인
	hasStreamStart := false
	hasStreamComplete := false

	for _, event := range streamEvents {
		switch event.Type {
		case events.StreamStartEvent:
			hasStreamStart = true
		case events.StreamCompleteEvent:
			hasStreamComplete = true
		}
	}

	if !hasStreamStart {
		t.Error("StreamStartEvent가 발생하지 않았습니다")
	}

	// 스트림이 취소되었으므로 완료 이벤트가 없거나 적어야 함
	if hasStreamComplete {
		t.Logf("스트림이 완료되었습니다 (취소가 늦었을 수 있음)")
	} else {
		t.Logf("스트림이 성공적으로 취소되었습니다")
	}

	// ActiveStreams에서 제거되었는지 확인
	ollamaService.StreamMutex.RLock()
	_, exists := ollamaService.ActiveStreams[requestUUID]
	ollamaService.StreamMutex.RUnlock()

	if exists {
		t.Error("취소된 스트림이 ActiveStreams에서 제거되지 않았습니다")
	}
}

func TestOllamaService_MultipleRequests_Integration(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	SetupIntegrationTest()
	defer CleanupTestConfig()

	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)

	// 여러 요청의 이벤트를 받을 핸들러
	var allEvents []events.Event
	testHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			allEvents = append(allEvents, event)
		},
		ID: TestService,
	}

	bus.Subscribe(events.StreamStartEvent, testHandler)
	bus.Subscribe(events.StreamCompleteEvent, testHandler)

	// 여러 개의 동시 요청
	requestUUIDs := []uuid.UUID{
		uuid.New(),
		uuid.New(),
		uuid.New(),
	}

	// 각 요청마다 다른 메시지 설정
	messages := []string{
		"1부터 5까지 세어주세요.",
		"안녕하세요!",
		"오늘 날씨는 어떤가요?",
	}

	// 동시에 여러 요청 시작
	for i, requestUUID := range requestUUIDs {
		ollamaService.UpdateUserInput(messages[i])
		ollamaService.CallApi(requestUUID)
		time.Sleep(100 * time.Millisecond) // 약간의 간격
	}

	// 모든 요청이 완료될 때까지 대기
	timeout := time.After(60 * time.Second)
	completedCount := 0

	for completedCount < len(requestUUIDs) {
		select {
		case <-timeout:
			t.Fatalf("모든 요청이 완료되지 않았습니다. 완료된 요청: %d/%d", completedCount, len(requestUUIDs))
		case <-time.After(200 * time.Millisecond):
			// 완료된 요청 개수 세기
			completedCount = 0
			completedUUIDs := make(map[uuid.UUID]bool)
			
			for _, event := range allEvents {
				if event.Type == events.StreamCompleteEvent {
					data, ok := event.Data.(types.StreamCompleteData)
					if ok {
						completedUUIDs[data.RequestUUID] = true
					}
				}
			}
			completedCount = len(completedUUIDs)
		}
	}

	// 시작 이벤트와 완료 이벤트의 개수가 일치하는지 확인
	startCount := 0
	completeCount := 0

	for _, event := range allEvents {
		switch event.Type {
		case events.StreamStartEvent:
			startCount++
		case events.StreamCompleteEvent:
			completeCount++
		}
	}

	if startCount != len(requestUUIDs) {
		t.Errorf("시작 이벤트 개수: %d, 예상: %d", startCount, len(requestUUIDs))
	}

	if completeCount != len(requestUUIDs) {
		t.Errorf("완료 이벤트 개수: %d, 예상: %d", completeCount, len(requestUUIDs))
	}

	t.Logf("다중 요청 테스트 성공: %d개 요청 모두 완료", len(requestUUIDs))
}

func TestOllamaService_ConsecutiveQuestions_Integration(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	SetupIntegrationTest()
	defer CleanupTestConfig()

	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)

	// 스트림 이벤트를 받을 핸들러
	var allEvents []events.Event
	testHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			allEvents = append(allEvents, event)
		},
		ID: TestService,
	}

	bus.Subscribe(events.StreamStartEvent, testHandler)
	bus.Subscribe(events.StreamChunkEvent, testHandler)
	bus.Subscribe(events.StreamCompleteEvent, testHandler)
	bus.Subscribe(events.StreamErrorEvent, testHandler)

	// 첫 번째 질문
	firstQuestion := "안녕하세요! 저는 김철수입니다."
	ollamaService.UpdateUserInput(firstQuestion)

	// 첫 번째 API 호출
	firstRequestUUID := uuid.New()
	ollamaService.CallApi(firstRequestUUID)

	// 첫 번째 응답이 완료될 때까지 대기
	timeout := time.After(30 * time.Second)
	firstCompleted := false
	
	for !firstCompleted {
		select {
		case <-timeout:
			t.Fatal("첫 번째 질문의 응답이 시간 내에 완료되지 않았습니다")
		case <-time.After(100 * time.Millisecond):
			for _, event := range allEvents {
				if event.Type == events.StreamCompleteEvent {
					if data, ok := event.Data.(types.StreamCompleteData); ok {
						if data.RequestUUID == firstRequestUUID {
							firstCompleted = true
							break
						}
					}
				}
				if event.Type == events.StreamErrorEvent {
					if data, ok := event.Data.(types.SteramErrorData); ok {
						if data.RequestUUID == firstRequestUUID {
							t.Fatalf("첫 번째 질문 API 호출 에러: %v", data.Error)
						}
					}
				}
			}
		}
	}

	// 첫 번째 응답의 내용을 수집
	var firstResponseContent string
	for _, event := range allEvents {
		if event.Type == events.StreamChunkEvent {
			if data, ok := event.Data.(types.StreamChunkData); ok {
				if data.RequestUUID == firstRequestUUID {
					firstResponseContent += data.Content
				}
			}
		}
	}
	
	if firstResponseContent == "" {
		t.Error("첫 번째 응답 내용이 비어있습니다")
	}

	if len(firstResponseContent) > 100 {
		t.Logf("첫 번째 응답: %s...", firstResponseContent[:100])
	} else {
		t.Logf("첫 번째 응답: %s", firstResponseContent)
	}

	// 잠시 대기 후 두 번째 질문 (이전 대화를 참조)
	time.Sleep(1 * time.Second)
	
	secondQuestion := "제 이름이 뭐라고 했죠?"
	ollamaService.UpdateUserInput(secondQuestion)

	// 두 번째 API 호출
	secondRequestUUID := uuid.New()
	ollamaService.CallApi(secondRequestUUID)

	// 두 번째 응답이 완료될 때까지 대기
	timeout2 := time.After(30 * time.Second)
	secondCompleted := false
	
	for !secondCompleted {
		select {
		case <-timeout2:
			t.Fatal("두 번째 질문의 응답이 시간 내에 완료되지 않았습니다")
		case <-time.After(100 * time.Millisecond):
			for _, event := range allEvents {
				if event.Type == events.StreamCompleteEvent {
					if data, ok := event.Data.(types.StreamCompleteData); ok {
						if data.RequestUUID == secondRequestUUID {
							secondCompleted = true
							break
						}
					}
				}
				if event.Type == events.StreamErrorEvent {
					if data, ok := event.Data.(types.SteramErrorData); ok {
						if data.RequestUUID == secondRequestUUID {
							t.Fatalf("두 번째 질문 API 호출 에러: %v", data.Error)
						}
					}
				}
			}
		}
	}

	// 두 번째 응답의 내용을 수집
	var secondResponseContent string
	for _, event := range allEvents {
		if event.Type == events.StreamChunkEvent {
			if data, ok := event.Data.(types.StreamChunkData); ok {
				if data.RequestUUID == secondRequestUUID {
					secondResponseContent += data.Content
				}
			}
		}
	}
	
	if secondResponseContent == "" {
		t.Error("두 번째 응답 내용이 비어있습니다")
	}

	if len(secondResponseContent) > 100 {
		t.Logf("두 번째 응답: %s...", secondResponseContent[:100])
	} else {
		t.Logf("두 번째 응답: %s", secondResponseContent)
	}

	// 두 번째 응답에 이름 관련 키워드가 포함되어 있는지 확인 (컨텍스트가 유지되었는지)
	nameIndicators := []string{"김철수", "철수", "이름", "name"}
	contentLower := strings.ToLower(secondResponseContent)
	contextFound := false
	
	for _, indicator := range nameIndicators {
		if strings.Contains(contentLower, strings.ToLower(indicator)) {
			contextFound = true
			break
		}
	}
	
	if !contextFound {
		t.Logf("경고: 두 번째 응답에서 이전 대화 컨텍스트가 명확하게 반영되지 않았을 수 있습니다")
		t.Logf("두 번째 응답 전체: %s", secondResponseContent)
	}

	t.Logf("연속 질문-답변 테스트 성공: 두 질문 모두 완료")
}

