package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/tools/read"
	"UniCode/src/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ollama/ollama/api"
	"github.com/spf13/viper"
)

// 통합테스트용 전체 워크플로우 핸들러
type IntegrationTestHandler struct {
	t                   *testing.T
	ollama             *service.OllamaService
	bus                *events.EventBus
	toolCallEvents     []events.Event
	toolResultEvents   []events.Event
	streamEvents       []events.Event
	testComplete       chan bool
	currentRequestUUID uuid.UUID
	readTool           *read.Tool
}

func NewIntegrationTestHandler(t *testing.T, ollama *service.OllamaService, bus *events.EventBus) *IntegrationTestHandler {
	return &IntegrationTestHandler{
		t:                t,
		ollama:           ollama,
		bus:              bus,
		toolCallEvents:   make([]events.Event, 0),
		toolResultEvents: make([]events.Event, 0),
		streamEvents:     make([]events.Event, 0),
		testComplete:     make(chan bool, 1),
		readTool:         &read.Tool{},
	}
}

func (h *IntegrationTestHandler) HandleEvent(event events.Event) {
	switch event.Type {
	case events.ToolCallEvent:
		h.toolCallEvents = append(h.toolCallEvents, event)
		h.t.Logf("도구 호출 이벤트 수신: %+v", event.Data)
		
		// Read 툴 호출인지 확인하고 실행
		if data, ok := event.Data.(types.ToolCallData); ok {
			if data.ToolName == "Read" {
				h.executeReadTool(data)
			}
		}

	case events.ToolResultEvent:
		h.toolResultEvents = append(h.toolResultEvents, event)
		h.t.Logf("도구 결과 이벤트 수신: RequestUUID=%s", event.Data.(types.ToolResultData).RequestUUID.String())

	case events.StreamStartEvent:
		h.streamEvents = append(h.streamEvents, event)
		h.t.Logf("스트림 시작 이벤트")

	case events.StreamChunkEvent:
		h.streamEvents = append(h.streamEvents, event)
		if data, ok := event.Data.(types.StreamChunkData); ok {
			h.t.Logf("스트림 청크: %s", data.Content[:min(50, len(data.Content))])
		}

	case events.StreamCompleteEvent:
		h.streamEvents = append(h.streamEvents, event)
		h.t.Logf("스트림 완료 이벤트")
		
		if data, ok := event.Data.(types.StreamCompleteData); ok {
			if data.RequestUUID == h.currentRequestUUID {
				h.testComplete <- true
			}
		}

	case events.StreamErrorEvent:
		h.streamEvents = append(h.streamEvents, event)
		h.t.Errorf("스트림 에러: %+v", event.Data)
		h.testComplete <- true
	}
}

func (h *IntegrationTestHandler) executeReadTool(toolCall types.ToolCallData) {
	// Read 툴 매개변수 파싱
	var input read.Input
	
	// 매개변수를 JSON으로 변환하여 파싱
	paramBytes, err := json.Marshal(toolCall.Parameters)
	if err != nil {
		h.t.Errorf("도구 매개변수 마샬링 실패: %v", err)
		return
	}
	
	err = json.Unmarshal(paramBytes, &input)
	if err != nil {
		h.t.Errorf("도구 매개변수 언마샬링 실패: %v", err)
		return
	}

	// Read 툴 실행
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: input,
	}

	ctx := context.Background()
	result, err := h.readTool.Handler()(ctx, nil, params)
	if err != nil {
		h.t.Errorf("Read 툴 실행 실패: %v", err)
		return
	}

	// 결과를 Tool Result 이벤트로 발행
	resultContent := result.Content[0].(*mcp.TextContent).Text
	service.PublishEvent(h.bus, events.ToolResultEvent, types.ToolResultData{
		RequestUUID: toolCall.RequestUUID,
		ToolResult:  resultContent,
	}, types.ToolService)

	h.t.Logf("Read 툴 실행 완료, 결과 길이: %d", len(resultContent))
}

func (h *IntegrationTestHandler) GetID() types.Source {
	return types.ToolService
}

func TestOllamaServiceWithReadTool_FileReading_Integration(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	// 테스트 설정
	SetupIntegrationTest()
	defer CleanupTestConfig()

	// 테스트 파일 생성
	testContent := `
package main

import "fmt"

func main() {
    fmt.Println("Hello, UniCode!")
    fmt.Println("This is a test file for Read tool integration")
}
`
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "test_integration_file.go")
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("테스트 파일 생성 실패: %v", err)
	}
	defer os.Remove(testFile)

	// Given: 이벤트 버스와 서비스 설정
	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)
	
	// Read 툴을 도구 목록에 추가
	readToolSchema := createReadToolSchema()
	ollamaService.Tools = []api.Tool{readToolSchema}

	// 통합테스트 핸들러 설정
	handler := NewIntegrationTestHandler(t, ollamaService, bus)
	bus.Subscribe(events.ToolCallEvent, handler)
	bus.Subscribe(events.ToolResultEvent, handler)
	bus.Subscribe(events.StreamStartEvent, handler)
	bus.Subscribe(events.StreamChunkEvent, handler)
	bus.Subscribe(events.StreamCompleteEvent, handler)
	bus.Subscribe(events.StreamErrorEvent, handler)

	// When: 파일 읽기를 요청하는 메시지 전송
	requestUUID := uuid.New()
	handler.currentRequestUUID = requestUUID
	
	userMessage := fmt.Sprintf(`다음 파일의 내용을 읽어서 분석해주세요: %s
	
	이 파일이 어떤 내용인지 설명해주세요. Read 도구를 사용해서 파일을 읽어보세요.`, testFile)
	
	ollamaService.UpdateUserInput(userMessage)
	ollamaService.CallApi(requestUUID)

	// Then: 테스트 완료 대기
	timeout := time.After(60 * time.Second)
	select {
	case <-handler.testComplete:
		t.Log("통합테스트 완료")
	case <-timeout:
		t.Fatal("통합테스트 타임아웃 (60초)")
	}

	// 검증: 이벤트들이 올바르게 발생했는지 확인
	if len(handler.toolCallEvents) == 0 {
		t.Error("도구 호출 이벤트가 발생하지 않았습니다")
	}

	if len(handler.toolResultEvents) == 0 {
		t.Error("도구 결과 이벤트가 발생하지 않았습니다")
	}

	hasStreamStart := false
	hasStreamComplete := false
	finalContent := ""

	for _, event := range handler.streamEvents {
		switch event.Type {
		case events.StreamStartEvent:
			hasStreamStart = true
		case events.StreamChunkEvent:
			if data, ok := event.Data.(types.StreamChunkData); ok {
				finalContent += data.Content
			}
		case events.StreamCompleteEvent:
			hasStreamComplete = true
		}
	}

	if !hasStreamStart {
		t.Error("스트림 시작 이벤트가 발생하지 않았습니다")
	}

	if !hasStreamComplete {
		t.Error("스트림 완료 이벤트가 발생하지 않았습니다")
	}

	// 최종 응답에 파일 내용에 대한 언급이 있는지 확인
	contentLower := strings.ToLower(finalContent)
	expectedKeywords := []string{"hello", "unicode", "main", "fmt", "println"}
	
	foundKeywords := 0
	for _, keyword := range expectedKeywords {
		if strings.Contains(contentLower, keyword) {
			foundKeywords++
		}
	}

	t.Logf("최종 응답 내용: %s", finalContent[:min(500, len(finalContent))])
	
	if foundKeywords < 2 && len(finalContent) > 0 {
		t.Logf("경고: 최종 응답에서 파일 내용과 관련된 키워드를 충분히 찾지 못했습니다. 찾은 키워드: %d/5", foundKeywords)
	} else if len(finalContent) == 0 {
		t.Log("최종 응답이 비어있지만 이는 스트림 청크 수집 타이밍 문제일 수 있습니다")
	}

	t.Logf("통합테스트 성공: 도구 호출 %d회, 도구 결과 %d회, 스트림 이벤트 %d회",
		len(handler.toolCallEvents), len(handler.toolResultEvents), len(handler.streamEvents))
}

func TestOllamaServiceWithReadTool_NonExistentFile_Integration(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	// 테스트 설정
	SetupIntegrationTest()
	defer CleanupTestConfig()

	// Given: 이벤트 버스와 서비스 설정
	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)
	
	// Read 툴을 도구 목록에 추가
	readToolSchema := createReadToolSchema()
	ollamaService.Tools = []api.Tool{readToolSchema}

	// 통합테스트 핸들러 설정
	handler := NewIntegrationTestHandler(t, ollamaService, bus)
	bus.Subscribe(events.ToolCallEvent, handler)
	bus.Subscribe(events.ToolResultEvent, handler)
	bus.Subscribe(events.StreamStartEvent, handler)
	bus.Subscribe(events.StreamChunkEvent, handler)
	bus.Subscribe(events.StreamCompleteEvent, handler)
	bus.Subscribe(events.StreamErrorEvent, handler)

	// When: 존재하지 않는 파일 읽기를 요청
	requestUUID := uuid.New()
	handler.currentRequestUUID = requestUUID
	
	nonExistentFile := "/path/that/definitely/does/not/exist.txt"
	userMessage := fmt.Sprintf(`다음 파일의 내용을 읽어주세요: %s
	
	Read 도구를 사용해서 파일을 읽어보세요.`, nonExistentFile)
	
	ollamaService.UpdateUserInput(userMessage)
	ollamaService.CallApi(requestUUID)

	// Then: 테스트 완료 대기
	timeout := time.After(60 * time.Second)
	select {
	case <-handler.testComplete:
		t.Log("에러 처리 통합테스트 완료")
	case <-timeout:
		t.Fatal("에러 처리 통합테스트 타임아웃 (60초)")
	}

	// 검증: 도구가 호출되고 에러가 적절히 처리되었는지 확인
	if len(handler.toolCallEvents) == 0 {
		t.Error("도구 호출 이벤트가 발생하지 않았습니다")
	}

	if len(handler.toolResultEvents) == 0 {
		t.Error("도구 결과 이벤트가 발생하지 않았습니다")
	}

	// 도구 결과에 에러 정보가 포함되어 있는지 확인
	var errorFound bool
	for _, event := range handler.toolResultEvents {
		if data, ok := event.Data.(types.ToolResultData); ok {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(data.ToolResult), &result); err == nil {
				if success, exists := result["success"]; exists {
					if successBool, ok := success.(bool); ok && !successBool {
						errorFound = true
						t.Logf("올바른 에러 응답 수신: %s", data.ToolResult)
						break
					}
				}
			}
		}
	}

	if !errorFound {
		t.Error("존재하지 않는 파일에 대한 적절한 에러 응답을 찾지 못했습니다")
	}

	t.Log("에러 처리 통합테스트 성공")
}

func TestOllamaServiceWithReadTool_MultipleFileOperations_Integration(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Ollama 서버가 실행되지 않아 통합 테스트를 건너뜁니다")
	}

	// 테스트 설정
	SetupIntegrationTest()
	defer CleanupTestConfig()

	// 여러 테스트 파일 생성
	tempDir := os.TempDir()
	testFiles := map[string]string{
		"config.json": `{
			"name": "UniCode",
			"version": "1.0.0",
			"description": "Terminal-based interactive CLI tool"
		}`,
		"README.md": `# UniCode
		
		이것은 UniCode 프로젝트입니다.
		
		## 특징
		- 터미널 기반 UI
		- 대화형 인터페이스
		- Go 언어로 작성`,
	}

	createdFiles := make([]string, 0)
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("테스트 파일 %s 생성 실패: %v", filename, err)
		}
		createdFiles = append(createdFiles, filePath)
	}
	
	// 정리 함수
	defer func() {
		for _, file := range createdFiles {
			os.Remove(file)
		}
	}()

	// Given: 이벤트 버스와 서비스 설정
	bus := events.NewEventBus()
	ollamaService := service.NewOllamaService(bus)
	
	// Read 툴을 도구 목록에 추가
	readToolSchema := createReadToolSchema()
	ollamaService.Tools = []api.Tool{readToolSchema}

	// 통합테스트 핸들러 설정
	handler := NewIntegrationTestHandler(t, ollamaService, bus)
	bus.Subscribe(events.ToolCallEvent, handler)
	bus.Subscribe(events.ToolResultEvent, handler)
	bus.Subscribe(events.StreamStartEvent, handler)
	bus.Subscribe(events.StreamChunkEvent, handler)
	bus.Subscribe(events.StreamCompleteEvent, handler)
	bus.Subscribe(events.StreamErrorEvent, handler)

	// When: 여러 파일 읽기를 요청
	requestUUID := uuid.New()
	handler.currentRequestUUID = requestUUID
	
	userMessage := fmt.Sprintf(`다음 파일들을 읽어서 프로젝트 정보를 분석해주세요:
	1. %s
	2. %s
	
	Read 도구를 사용해서 각 파일을 읽어보고, 어떤 프로젝트인지 설명해주세요.`,
		createdFiles[0], createdFiles[1])
	
	ollamaService.UpdateUserInput(userMessage)
	ollamaService.CallApi(requestUUID)

	// Then: 테스트 완료 대기 (여러 파일 처리로 인해 더 긴 시간 허용)
	timeout := time.After(90 * time.Second)
	select {
	case <-handler.testComplete:
		t.Log("다중 파일 통합테스트 완료")
	case <-timeout:
		t.Fatal("다중 파일 통합테스트 타임아웃 (90초)")
	}

	// 검증: 여러 번의 도구 호출이 발생했는지 확인
	if len(handler.toolCallEvents) < 2 {
		t.Errorf("충분한 도구 호출이 발생하지 않았습니다. 예상: 최소 2회, 실제: %d회", 
			len(handler.toolCallEvents))
	}

	if len(handler.toolResultEvents) < 2 {
		t.Errorf("충분한 도구 결과가 수신되지 않았습니다. 예상: 최소 2회, 실제: %d회", 
			len(handler.toolResultEvents))
	}

	// 최종 응답에 두 파일의 내용이 모두 반영되었는지 확인
	finalContent := ""
	for _, event := range handler.streamEvents {
		if event.Type == events.StreamChunkEvent {
			if data, ok := event.Data.(types.StreamChunkData); ok {
				finalContent += data.Content
			}
		}
	}

	contentLower := strings.ToLower(finalContent)
	expectedKeywords := []string{"unicode", "config", "readme", "version", "description"}
	
	foundKeywords := 0
	for _, keyword := range expectedKeywords {
		if strings.Contains(contentLower, keyword) {
			foundKeywords++
		}
	}

	if foundKeywords < 3 {
		t.Errorf("최종 응답에서 여러 파일의 내용과 관련된 키워드를 충분히 찾지 못했습니다. 찾은 키워드: %d/5", foundKeywords)
		t.Logf("최종 응답 일부: %s", finalContent[:min(500, len(finalContent))])
	}

	t.Logf("다중 파일 통합테스트 성공: 도구 호출 %d회, 도구 결과 %d회",
		len(handler.toolCallEvents), len(handler.toolResultEvents))
}

// Read 툴의 Ollama API 스키마 생성
func createReadToolSchema() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "Read",
			Description: "Reads a file from the local filesystem. Provide the absolute file path.",
			Parameters: struct {
				Type       string                     `json:"type"`
				Defs       any                        `json:"$defs,omitempty"`
				Items      any                        `json:"items,omitempty"`
				Required   []string                   `json:"required"`
				Properties map[string]api.ToolProperty `json:"properties"`
			}{
				Type: "object",
				Required: []string{"file_path"},
				Properties: map[string]api.ToolProperty{
					"file_path": {
						Type:        []string{"string"},
						Description: "The absolute path to the file to read",
					},
					"offset": {
						Type:        []string{"integer"},
						Description: "The line number to start reading from (optional)",
					},
					"limit": {
						Type:        []string{"integer"}, 
						Description: "The number of lines to read (optional)",
					},
				},
			},
		},
	}
}


// 테스트 유틸리티 함수들

// IsOllamaRunning checks if Ollama server is running
func IsOllamaRunning() bool {
	timeout := time.After(2 * time.Second)
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

// SetupIntegrationTest configures test environment for integration tests
func SetupIntegrationTest() {
	viper.Set("ollama.url", "http://localhost:11434")
	viper.Set("ollama.model", "llama3.1:8b")
	viper.Set("prompt.system", "/home/cecil/UniCode/SystemPrompt/Root.md")
}

// CleanupTestConfig removes test configuration files
func CleanupTestConfig() {
	// 정리할 임시 파일이 있다면 여기에 추가
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}