package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/types"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
)

// 모든 스트림 출력을 자세히 분석하는 테스트
func TestAllStreamOutputAnalysis(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	SetupIntegrationTest()
	defer CleanupTestConfig()

	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)
	messageService := service.NewMessageService(bus)

	// 모든 이벤트를 수집하는 분석기
	var allEvents []events.Event
	var rawChunks []string
	var completeMessages []string
	var toolCalls []types.ToolCallData

	analysisHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			allEvents = append(allEvents, event)
				
				switch event.Type {
				case events.StreamChunkEvent:
					if data, ok := event.Data.(types.StreamChunkData); ok {
						rawChunks = append(rawChunks, data.Content)
						t.Logf("청크: '%s' (완료: %t)", data.Content, data.IsComplete)
					}
				case events.StreamCompleteEvent:
					if data, ok := event.Data.(types.StreamCompleteData); ok {
						completeMessages = append(completeMessages, data.FinalMessage.Content)
						t.Logf("완료된 메시지: '%s'", data.FinalMessage.Content)
						t.Logf("도구 호출 개수: %d", len(data.FinalMessage.ToolCalls))
					}
				case events.ToolCallEvent:
					if data, ok := event.Data.(types.ToolCallData); ok {
						toolCalls = append(toolCalls, data)
						t.Logf("도구 호출: %s, 매개변수: %+v", data.ToolName, data.Parameters)
					}
				case events.StreamStartEvent:
				if data, ok := event.Data.(types.StreamStartData); ok {
					t.Logf("스트림 시작: %s", data.RequestUUID)
				}
			case events.StreamErrorEvent:
				if data, ok := event.Data.(types.StreamErrorData); ok {
					t.Logf("스트림 에러: %v", data.Error)
				}
			}
		},
		ID: TestService,
	}

	// 모든 관련 이벤트 구독
	bus.Subscribe(events.StreamStartEvent, analysisHandler)
	bus.Subscribe(events.StreamChunkEvent, analysisHandler)
	bus.Subscribe(events.StreamCompleteEvent, analysisHandler)
	bus.Subscribe(events.StreamErrorEvent, analysisHandler)
	bus.Subscribe(events.ToolCallEvent, analysisHandler)

	// 테스트 케이스들
	testCases := []struct {
		name    string
		message string
		expectToolCalls bool
	}{
		{
			name:    "간단한 인사",
			message: "안녕하세요! 간단히 인사해주세요.",
			expectToolCalls: false,
		},
		{
			name:    "긴 응답 요청",
			message: "프로그래밍의 역사에 대해 자세히 설명해주세요.",
			expectToolCalls: false,
		},
		{
			name:    "숫자 계산",
			message: "1부터 10까지의 합을 계산해주세요.",
			expectToolCalls: false,
		},
		{
			name:    "간단한 Go 코드 출력",
			message: "Go 언어로 Hello World를 출력하는 간단한 코드를 작성해주세요.",
			expectToolCalls: false,
		},
		{
			name:    "함수 예제 코드",
			message: "Go에서 덧셈을 하는 함수를 작성해주세요.",
			expectToolCalls: false,
		},
		{
			name:    "복잡한 코드 구조",
			message: "Go에서 구조체와 메서드를 사용하는 예제를 보여주세요. 주석도 포함해서 작성해주세요.",
			expectToolCalls: false,
		},
		{
			name: "마크다운 문서 출력",
			message: "간단한 마크다운 문서를 작성해주세요.",
			expectToolCalls: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 이전 데이터 초기화
			allEvents = nil
			rawChunks = nil
			completeMessages = nil
			toolCalls = nil

			// OllamaService의 Messages 초기화 (이전 대화 히스토리 제거)
			ollamaService.Messages = make([]api.Message, 0, 100)

			// 메시지 업데이트 및 API 호출
			ollamaService.UpdateUserInput(tc.message)
			requestUUID := uuid.New()
			ollamaService.CallApi(requestUUID)

			// 응답 완료 대기
			timeout := time.After(IntegrationTimeout)
			completed := false

			for !completed {
				select {
				case <-timeout:
					t.Fatal("API 호출 시간 초과")
				case <-time.After(AsyncWaitTime):
					for _, event := range allEvents {
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

			// 상세 분석
			t.Logf("=== %s 분석 결과 ===", tc.name)
			t.Logf("총 이벤트 수: %d", len(allEvents))
			t.Logf("청크 수: %d", len(rawChunks))
			t.Logf("완료 메시지 수: %d", len(completeMessages))
			t.Logf("도구 호출 수: %d", len(toolCalls))

			// 청크들을 조합해서 완전한 메시지 구성
			fullMessage := strings.Join(rawChunks, "")
			t.Logf("청크로 조합된 전체 메시지: '%s'", fullMessage)

			// 완료 메시지와 청크 조합 비교
			if len(completeMessages) > 0 {
				finalMessage := completeMessages[len(completeMessages)-1]
				t.Logf("최종 완료 메시지: '%s'", finalMessage)
				
				// 메시지 파싱이 필요한지 확인
				needsParsing := checkIfMessageNeedsParsing(fullMessage, finalMessage)
				t.Logf("메시지 파싱 필요 여부: %t", needsParsing)
				
				// 코드 블록 분석
				codeAnalysis := analyzeCodeContent(fullMessage)
				t.Logf("코드 분석 결과: %s", codeAnalysis)
				
				if needsParsing {
					t.Logf("MessageService가 필요할 수 있습니다 - 청크와 최종 메시지가 다릅니다")
				} else {
					t.Logf("MessageService가 불필요할 수 있습니다 - 청크와 최종 메시지가 동일합니다")
				}
			}

			// 도구 호출 예상과 실제 비교
			if tc.expectToolCalls && len(toolCalls) == 0 {
				t.Logf("경고: 도구 호출이 예상되었지만 발생하지 않았습니다")
			} else if !tc.expectToolCalls && len(toolCalls) > 0 {
				t.Logf("예상치 못한 도구 호출이 발생했습니다")
			}

			// 이벤트 타임라인 출력
			t.Logf("=== 이벤트 타임라인 ===")
			for i, event := range allEvents {
				t.Logf("%d. %v at %s", i+1, event.Type, event.Timestamp.Format(time.RFC3339Nano))
			}
		})
	}

	// MessageService 사용 권장사항 출력
	recommendMessageServiceUsage(t, messageService)
}

// 메시지 파싱이 필요한지 확인하는 함수
func checkIfMessageNeedsParsing(chunkMessage, finalMessage string) bool {
	// 공백 정규화
	chunkMessage = strings.TrimSpace(chunkMessage)
	finalMessage = strings.TrimSpace(finalMessage)
	
	// 길이 비교
	if len(chunkMessage) != len(finalMessage) {
		return true
	}
	
	// 내용 비교
	if chunkMessage != finalMessage {
		return true
	}
	
	// 특수 문자나 포매팅 확인
	hasSpecialChars := strings.Contains(finalMessage, "```") || 
					  strings.Contains(finalMessage, "**") ||
					  strings.Contains(finalMessage, "*") ||
					  strings.Contains(finalMessage, "_") ||
					  strings.Contains(finalMessage, "[") ||
					  strings.Contains(finalMessage, "]")
	
	return hasSpecialChars
}

// 코드 내용 분석 함수
func analyzeCodeContent(content string) string {
	analysis := []string{}
	
	// 마크다운 코드 블록 감지
	codeBlockCount := strings.Count(content, "```")
	if codeBlockCount >= 2 {
		analysis = append(analysis, fmt.Sprintf("마크다운 코드 블록 %d개 감지", codeBlockCount/2))
	}
	
	// 인라인 코드 감지
	inlineCodeCount := strings.Count(content, "`") - codeBlockCount
	if inlineCodeCount > 0 {
		analysis = append(analysis, fmt.Sprintf("인라인 코드 %d개", inlineCodeCount/2))
	}
	
	// Go 키워드 감지
	goKeywords := []string{"func", "package", "import", "var", "const", "type", "struct", "interface"}
	for _, keyword := range goKeywords {
		if strings.Contains(content, keyword) {
			analysis = append(analysis, fmt.Sprintf("Go 키워드 '%s' 포함", keyword))
			break
		}
	}
	
	// 주석 감지
	if strings.Contains(content, "//") || strings.Contains(content, "/*") {
		analysis = append(analysis, "코드 주석 포함")
	}
	
	// 함수 정의 감지
	if strings.Contains(content, "func ") {
		analysis = append(analysis, "함수 정의 포함")
	}
	
	// 구조체 정의 감지
	if strings.Contains(content, "type ") && strings.Contains(content, "struct") {
		analysis = append(analysis, "구조체 정의 포함")
	}
	
	if len(analysis) == 0 {
		return "일반 텍스트 (코드 없음)"
	}
	
	return strings.Join(analysis, ", ")
}

// MessageService 사용 권장사항을 분석하는 함수
func recommendMessageServiceUsage(t *testing.T, _ *service.MessageService) {
	t.Logf("=== MessageService 분석 결과 ===")
	
	// 현재 MessageService는 빈 구현체
	t.Logf("현재 MessageService.ParingMessage는 빈 구현체입니다")
	t.Logf("MessageService.HandleEvent는 StreamStartEvent만 구독하지만 처리 로직이 없습니다")
	
	// 권장사항
	t.Logf("=== 권장사항 ===")
	t.Logf("1. 만약 스트림 청크와 최종 메시지가 항상 동일하다면 MessageService는 불필요할 수 있습니다")
	t.Logf("2. 만약 메시지에 마크다운이나 특수 포매팅이 포함된다면 MessageService가 유용할 수 있습니다")
	t.Logf("3. 현재 OllamaService가 이미 적절한 이벤트들을 발생시키고 있습니다")
	t.Logf("4. MessageService를 제거하거나 실제 파싱 로직을 구현하는 것을 고려해보세요")
}

// 실제 메시지 파싱 성능 테스트
func TestMessageParsingPerformance(t *testing.T) {
	// 다양한 크기의 메시지로 파싱 성능 테스트
	testMessages := []string{
		"간단한 메시지",
		strings.Repeat("긴 메시지 테스트 ", 100),
		"```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
		"**볼드** 텍스트와 *이탤릭* 텍스트가 포함된 마크다운",
	}

	for i, msg := range testMessages {
		t.Run(fmt.Sprintf("메시지_%d", i+1), func(t *testing.T) {
			start := time.Now()
			
			// 가상의 파싱 작업 (실제로는 MessageService에서 구현)
			parsed := parseMessage(msg)
			
			duration := time.Since(start)
			t.Logf("메시지 길이: %d, 파싱 시간: %v, 결과: %s", len(msg), duration, parsed[:min(50, len(parsed))])
		})
	}
}

// 간단한 메시지 파싱 함수 (예시)
func parseMessage(content string) string {
	// 실제 MessageService에서 구현할 수 있는 파싱 로직 예시
	
	// 마크다운 코드 블록 감지
	if strings.Contains(content, "```") {
		return fmt.Sprintf("[코드 블록 포함] %s", content)
	}
	
	// 볼드/이탤릭 텍스트 감지
	if strings.Contains(content, "**") || strings.Contains(content, "*") {
		return fmt.Sprintf("[포매팅 포함] %s", content)
	}
	
	// 일반 텍스트
	return fmt.Sprintf("[일반 텍스트] %s", content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}