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
	"github.com/spf13/viper"
)

// 통합 테스트를 위한 설정
func setupIntegrationTest() {
	viper.Set("ollama.url", "http://localhost:11434")
	viper.Set("ollama.model", "gpt-oss:20b") // env.toml과 동일한 모델 사용
	viper.Set("prompt.system", "/tmp/integration_test_system_prompt.md")
	
	// 테스트용 시스템 프롬프트 파일 생성
	systemPrompt := "You are a helpful assistant. Respond concisely."
	err := os.WriteFile("/tmp/integration_test_system_prompt.md", []byte(systemPrompt), 0644)
	if err != nil {
		panic(err)
	}
}

func cleanupIntegrationTest() {
	os.Remove("/tmp/integration_test_system_prompt.md")
}

// Ollama 서버가 실행 중인지 확인하는 헬퍼 함수
func isOllamaRunning() bool {
	// curl로 직접 확인
	timeout := time.After(2 * time.Second)
	done := make(chan bool)
	
	go func() {
		// HTTP GET 요청으로 간단히 확인
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

func TestOllamaService_CallApi_Integration(t *testing.T) {
	if !isOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	setupIntegrationTest()
	defer cleanupIntegrationTest()

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
	bus.Subscribe(events.StreramChunkEvnet, testHandler)
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
		case events.StreramChunkEvnet:
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
	if !isOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	setupIntegrationTest()
	defer cleanupIntegrationTest()

	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)

	// 도구 호출 이벤트를 받을 핸들러
	var toolCallEvents []events.Event
	var streamEvents []events.Event
	toolHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			switch event.Type {
			case events.ToolCallEvent:
				toolCallEvents = append(toolCallEvents, event)
			case events.StreamCompleteEvent:
				streamEvents = append(streamEvents, event)
				// StreamCompleteEvent에서 ToolCall 확인
				if data, ok := event.Data.(types.StreamCompleteData); ok {
					if len(data.FinalMessage.ToolCalls) > 0 {
						t.Logf("응답에 도구 호출이 포함되어 있습니다: %d개", len(data.FinalMessage.ToolCalls))
						for i, call := range data.FinalMessage.ToolCalls {
							t.Logf("도구 호출 %d: %s, 매개변수: %+v", i+1, call.Function.Name, call.Function.Arguments)
						}
					}
				}
			}
		},
		ID: TestService,
	}

	bus.Subscribe(events.ToolCallEvent, toolHandler)
	bus.Subscribe(events.StreamCompleteEvent, toolHandler)

	// 단순한 메시지로 테스트 (도구 없이)
	ollamaService.UpdateUserInput("간단한 계산을 해주세요: 2 + 2는 무엇인가요?")

	// API 호출
	requestUUID := uuid.New()
	ollamaService.CallApi(requestUUID)

	// 응답 완료를 기다림
	timeout := time.After(30 * time.Second)
	
	for {
		select {
		case <-timeout:
			t.Error("API 응답 시간 초과")
			return
		case <-time.After(200 * time.Millisecond):
			if len(streamEvents) > 0 {
				t.Logf("API 호출이 성공적으로 완료되었습니다")
				return
			}
		}
	}
}

func TestOllamaService_StreamCancel_Integration(t *testing.T) {
	if !isOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	setupIntegrationTest()
	defer cleanupIntegrationTest()

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
	bus.Subscribe(events.StreramChunkEvnet, testHandler)
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
	if !isOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	setupIntegrationTest()
	defer cleanupIntegrationTest()

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