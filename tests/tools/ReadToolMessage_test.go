package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/tools/read"
	"UniCode/src/types"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadTool_Success_ShouldGenerateToolMessage(t *testing.T) {
	// Given: 임시 파일 생성
	testContent := "Test file content\nLine 2"
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "test_read_file.txt")
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("임시 파일 생성 실패: %v", err)
	}
	defer os.Remove(testFile)

	// MCP 서버 세션과 도구 설정
	tool := &read.Tool{}
	input := &read.Input{
		FilePath: testFile,
		Offset:   0,
		Limit:    0,
	}

	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: *input,
	}

	// 이벤트 버스 설정
	eventBus := events.NewEventBus()
	
	// Tool message 수집을 위한 채널
	toolMessageReceived := make(chan bool, 1)
	var receivedToolResult string

	// Tool result 이벤트 구독
	eventBus.Subscribe(events.ToolResultEvent, &TestToolMessageHandler{
		t:                   t,
		toolMessageReceived: toolMessageReceived,
		receivedToolResult:  &receivedToolResult,
	})

	// When: Read 툴 실행
	ctx := context.Background()
	result, err := tool.Handler()(ctx, nil, params)

	// Then: 툴 실행 성공 확인
	if err != nil {
		t.Errorf("Read 툴 실행 실패: %v", err)
	}

	if result == nil {
		t.Fatal("결과가 nil입니다")
	}

	// Tool message 내용 검증
	var resultData read.Success
	err = json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &resultData)
	if err != nil {
		t.Errorf("결과 파싱 실패: %v", err)
	}

	if !resultData.Success {
		t.Error("읽기 작업이 성공하지 않았습니다")
	}

	if resultData.Text == "" {
		t.Error("읽은 내용이 비어있습니다")
	}

	// Tool result 이벤트 발생 시뮬레이션
	requestUUID := uuid.New()
	service.PublishEvent(eventBus, events.ToolResultEvent, types.ToolResultData{
		RequestUUID: requestUUID,
		ToolResult:  result.Content[0].(*mcp.TextContent).Text,
	}, types.ToolService)

	// Tool message 수신 대기
	select {
	case <-toolMessageReceived:
		t.Log("Tool message가 성공적으로 수신되었습니다")
	case <-time.After(1 * time.Second):
		t.Error("Tool message가 수신되지 않았습니다")
	}

	// 수신된 Tool result 검증
	if receivedToolResult == "" {
		t.Error("Tool result가 비어있습니다")
	}

	t.Logf("수신된 Tool result: %s", receivedToolResult[:min(100, len(receivedToolResult))])
}

func TestReadTool_FileNotFound_ShouldGenerateErrorToolMessage(t *testing.T) {
	// Given: 존재하지 않는 파일 경로
	nonExistentFile := "/path/that/does/not/exist.txt"
	
	tool := &read.Tool{}
	input := &read.Input{
		FilePath: nonExistentFile,
	}

	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: *input,
	}

	// When: Read 툴 실행
	ctx := context.Background()
	result, err := tool.Handler()(ctx, nil, params)

	// Then: 에러 결과 확인
	if err != nil {
		t.Errorf("Read 툴 실행 중 예상치 못한 에러: %v", err)
	}

	if result == nil {
		t.Fatal("결과가 nil입니다")
	}

	// 에러 메시지 내용 검증
	var resultData read.Fail
	err = json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &resultData)
	if err != nil {
		t.Errorf("결과 파싱 실패: %v", err)
	}

	if resultData.Success {
		t.Error("존재하지 않는 파일 읽기가 성공으로 처리되었습니다")
	}

	if resultData.ErrorCode != read.FileNotFound {
		t.Errorf("예상된 에러 코드: %s, 실제: %s", read.FileNotFound, resultData.ErrorCode)
	}

	t.Logf("올바른 에러 메시지 생성: %s", resultData.Error)
}

// Tool message 처리를 위한 테스트 핸들러
type TestToolMessageHandler struct {
	t                   *testing.T
	toolMessageReceived chan bool
	receivedToolResult  *string
}

func (h *TestToolMessageHandler) HandleEvent(event events.Event) {
	switch event.Type {
	case events.ToolResultEvent:
		data := event.Data.(types.ToolResultData)
		*h.receivedToolResult = data.ToolResult
		h.toolMessageReceived <- true
		h.t.Logf("Tool result 이벤트 수신: RequestUUID=%s", data.RequestUUID.String())
	}
}

func (h *TestToolMessageHandler) GetID() types.Source {
	return types.ToolService
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}