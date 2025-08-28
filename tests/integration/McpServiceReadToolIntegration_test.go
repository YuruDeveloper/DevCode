package integration

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	mcpservice "DevCode/src/service/mcp"
	"DevCode/src/tools/read"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMcpServiceReadToolIntegration(t *testing.T) {
	// Given: EventBus와 MCP Service 통합 환경 설정
	logger := zap.NewNop()
	bus, err := events.NewEventBus(logger)
	require.NoError(t, err, "EventBus 생성 실패")

	// 테스트용 viper 설정
	setupTestViper()

	// 이벤트 수신을 위한 채널 설정
	toolResultEvents := make(chan events.Event, 10)

	// 이벤트 수신기
	eventSubscriber := &IntegrationEventSubscriber{
		toolResults: toolResultEvents,
	}

	bus.Subscribe(events.ToolRawResultEvent, eventSubscriber)

	// 임시 파일 생성 및 정리
	tmpDir, err := os.MkdirTemp("", "mcp_read_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("Read Tool 성공 시 올바른 이벤트 발행", func(t *testing.T) {
		// 테스트 파일 생성
		testFile := filepath.Join(tmpDir, "success_test.txt")
		testContent := "첫 번째 줄\n두 번째 줄\n세 번째 줄\n"
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// MCP Service 초기화 (실제 Read tool 포함)
		mcpService := mcpservice.NewMcpService(bus, logger)
		require.NotNil(t, mcpService)

		// When: Read tool 호출
		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "Read",
			Parameters: map[string]interface{}{
				"file_path": testFile,
			},
		}

		// AcceptToolEvent 발행
		acceptEvent := events.Event{
			Type:      events.AcceptToolEvent,
			Data:      toolCallData,
			Timestamp: time.Now(),
			Source:    constants.Model,
		}

		mcpService.HandleEvent(acceptEvent)

		// Then: 성공 결과 이벤트 확인
		select {
		case event := <-toolResultEvents:
			assert.Equal(t, events.ToolRawResultEvent, event.Type)
			assert.Equal(t, constants.McpService, event.Source)

			data, ok := event.Data.(dto.ToolRawResultData)
			require.True(t, ok, "이벤트 데이터 타입이 올바르지 않음")

			assert.Equal(t, toolCallData.RequestUUID, data.RequestUUID)
			assert.Equal(t, toolCallData.ToolCallUUID, data.ToolCall)
			assert.Nil(t, data.Error, "에러가 없어야 함")
			assert.NotNil(t, data.Result, "결과가 있어야 함")

			// Read tool 결과 내용 검증
			textContent, ok := data.Result.Content[0].(*mcp.TextContent)
			require.True(t, ok, "텍스트 컨텐츠여야 함")

			var readResult read.Success
			err := json.Unmarshal([]byte(textContent.Text), &readResult)
			require.NoError(t, err, "JSON 파싱 실패")

			assert.True(t, readResult.Success)
			assert.Equal(t, 3, readResult.TotalLines)
			assert.Equal(t, 3, readResult.LinesRead)
			assert.Contains(t, readResult.Text, "첫 번째 줄")
			assert.Contains(t, readResult.Text, "두 번째 줄")
			assert.Contains(t, readResult.Text, "세 번째 줄")

		case <-time.After(1 * time.Second):
			t.Fatal("성공 이벤트가 시간 내에 수신되지 않음")
		}
	})

	t.Run("Read Tool 파일 없음 에러 시 올바른 에러 이벤트 발행", func(t *testing.T) {
		// MCP Service 초기화
		mcpService := mcpservice.NewMcpService(bus, logger)
		require.NotNil(t, mcpService)

		// When: 존재하지 않는 파일로 Read tool 호출
		nonExistentFile := filepath.Join(tmpDir, "nonexistent_file.txt")
		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "Read",
			Parameters: map[string]interface{}{
				"file_path": nonExistentFile,
			},
		}

		// AcceptToolEvent 발행
		acceptEvent := events.Event{
			Type:      events.AcceptToolEvent,
			Data:      toolCallData,
			Timestamp: time.Now(),
			Source:    constants.Model,
		}

		mcpService.HandleEvent(acceptEvent)

		// Then: 에러 결과 이벤트 확인
		select {
		case event := <-toolResultEvents:
			assert.Equal(t, events.ToolRawResultEvent, event.Type)
			assert.Equal(t, constants.McpService, event.Source)

			data, ok := event.Data.(dto.ToolRawResultData)
			require.True(t, ok, "이벤트 데이터 타입이 올바르지 않음")

			assert.Equal(t, toolCallData.RequestUUID, data.RequestUUID)
			assert.Equal(t, toolCallData.ToolCallUUID, data.ToolCall)
			// MCP SDK에서는 에러 시에도 Result가 반환되고 IsError 플래그가 설정됨
			if data.Error != nil {
				// 에러가 Error 필드에 설정된 경우
				assert.Contains(t, data.Error.Error(), "file not found")
				assert.Contains(t, data.Error.Error(), nonExistentFile)
			} else if data.Result != nil && data.Result.IsError {
				// 에러가 Result의 IsError 플래그로 설정된 경우
				assert.True(t, data.Result.IsError, "IsError 플래그가 설정되어야 함")
				// Result 내용에서 에러 메시지 확인
				if len(data.Result.Content) > 0 {
					if textContent, ok := data.Result.Content[0].(*mcp.TextContent); ok {
						assert.Contains(t, textContent.Text, "file not found")
					}
				}
			} else {
				t.Fatal("에러 정보가 없음")
			}

		case <-time.After(1 * time.Second):
			t.Fatal("에러 이벤트가 시간 내에 수신되지 않음")
		}
	})

	t.Run("Read Tool 잘못된 파라미터 에러 시 올바른 에러 이벤트 발행", func(t *testing.T) {
		// MCP Service 초기화
		mcpService := mcpservice.NewMcpService(bus, logger)
		require.NotNil(t, mcpService)

		// When: 빈 파일 경로로 Read tool 호출
		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "Read",
			Parameters: map[string]interface{}{
				"file_path": "", // 빈 경로
			},
		}

		// AcceptToolEvent 발행
		acceptEvent := events.Event{
			Type:      events.AcceptToolEvent,
			Data:      toolCallData,
			Timestamp: time.Now(),
			Source:    constants.Model,
		}

		mcpService.HandleEvent(acceptEvent)

		// Then: 에러 결과 이벤트 확인
		select {
		case event := <-toolResultEvents:
			data := event.Data.(dto.ToolRawResultData)

			// 에러 처리 확인
			if data.Error != nil {
				assert.Contains(t, data.Error.Error(), "invalid path format")
			} else if data.Result != nil && data.Result.IsError {
				assert.True(t, data.Result.IsError, "IsError 플래그가 설정되어야 함")
			} else {
				t.Fatal("에러 정보가 없음")
			}

		case <-time.After(1 * time.Second):
			t.Fatal("에러 이벤트가 시간 내에 수신되지 않음")
		}
	})

	t.Run("Read Tool Offset과 Limit 파라미터 처리", func(t *testing.T) {
		// 테스트 파일 생성 (10줄)
		testFile := filepath.Join(tmpDir, "offset_limit_test.txt")
		var testContent string
		for i := 1; i <= 10; i++ {
			testContent += fmt.Sprintf("Line %d\n", i)
		}
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// MCP Service 초기화
		mcpService := mcpservice.NewMcpService(bus, logger)
		require.NotNil(t, mcpService)

		// When: Offset과 Limit을 사용한 Read tool 호출
		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "Read",
			Parameters: map[string]interface{}{
				"file_path": testFile,
				"offset":    3,  // 3번째 줄부터
				"limit":     3,  // 3줄만
			},
		}

		// AcceptToolEvent 발행
		acceptEvent := events.Event{
			Type:      events.AcceptToolEvent,
			Data:      toolCallData,
			Timestamp: time.Now(),
			Source:    constants.Model,
		}

		mcpService.HandleEvent(acceptEvent)

		// Then: 성공 결과 이벤트 확인
		select {
		case event := <-toolResultEvents:
			data := event.Data.(dto.ToolRawResultData)

			assert.Nil(t, data.Error, "에러가 없어야 함")
			assert.NotNil(t, data.Result, "결과가 있어야 함")

			// 결과 내용 검증
			textContent := data.Result.Content[0].(*mcp.TextContent)
			var readResult read.Success
			err := json.Unmarshal([]byte(textContent.Text), &readResult)
			require.NoError(t, err)

			assert.True(t, readResult.Success)
			assert.Equal(t, 10, readResult.TotalLines) // 전체 줄 수
			
			// Offset과 Limit이 적용되었는지 확인
			assert.Contains(t, readResult.Text, "     3\tLine 3")
			assert.Contains(t, readResult.Text, "     4\tLine 4")
			assert.Contains(t, readResult.Text, "     5\tLine 5")
			assert.NotContains(t, readResult.Text, "Line 1")
			assert.NotContains(t, readResult.Text, "Line 2")
			assert.NotContains(t, readResult.Text, "Line 6")

		case <-time.After(1 * time.Second):
			t.Fatal("성공 이벤트가 시간 내에 수신되지 않음")
		}
	})

	t.Run("Multiple Tool Calls 처리", func(t *testing.T) {
		// 두 개의 테스트 파일 생성
		testFile1 := filepath.Join(tmpDir, "multi_test1.txt")
		testFile2 := filepath.Join(tmpDir, "multi_test2.txt")
		
		err := os.WriteFile(testFile1, []byte("파일1 내용"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(testFile2, []byte("파일2 내용"), 0644)
		require.NoError(t, err)

		// MCP Service 초기화
		mcpService := mcpservice.NewMcpService(bus, logger)

		// When: 두 개의 Read tool을 연속 호출
		toolCallData1 := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "Read",
			Parameters: map[string]interface{}{
				"file_path": testFile1,
			},
		}

		toolCallData2 := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "Read",
			Parameters: map[string]interface{}{
				"file_path": testFile2,
			},
		}

		// 두 이벤트 연속 발행
		mcpService.HandleEvent(events.Event{
			Type:      events.AcceptToolEvent,
			Data:      toolCallData1,
			Timestamp: time.Now(),
			Source:    constants.Model,
		})

		mcpService.HandleEvent(events.Event{
			Type:      events.AcceptToolEvent,
			Data:      toolCallData2,
			Timestamp: time.Now(),
			Source:    constants.Model,
		})

		// Then: 두 결과 모두 수신 확인
		receivedResults := make(map[uuid.UUID]bool)
		
		for i := 0; i < 2; i++ {
			select {
			case event := <-toolResultEvents:
				data := event.Data.(dto.ToolRawResultData)
				receivedResults[data.ToolCall] = true

				assert.Nil(t, data.Error, "에러가 없어야 함")
				assert.NotNil(t, data.Result, "결과가 있어야 함")

			case <-time.After(2 * time.Second):
				t.Fatal("모든 이벤트가 시간 내에 수신되지 않음")
			}
		}

		assert.True(t, receivedResults[toolCallData1.ToolCallUUID], "첫 번째 tool call 결과 수신")
		assert.True(t, receivedResults[toolCallData2.ToolCallUUID], "두 번째 tool call 결과 수신")
	})
}

// 통합 테스트용 이벤트 수신기
type IntegrationEventSubscriber struct {
	toolResults chan events.Event
}

func (i *IntegrationEventSubscriber) HandleEvent(event events.Event) {
	switch event.Type {
	case events.ToolRawResultEvent:
		select {
		case i.toolResults <- event:
		default:
			// 채널이 가득 찬 경우 무시
		}
	}
}

func (i *IntegrationEventSubscriber) GetID() constants.Source {
	return constants.Model
}

// 테스트용 viper 설정
func setupTestViper() {
	viper.Set("mcp.name", "test-mcp")
	viper.Set("mcp.version", "1.0.0")
	viper.Set("server.name", "test-server")
	viper.Set("server.version", "1.0.0")
}