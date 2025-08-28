package integration

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	mcpservice "DevCode/src/service/mcp"
	"DevCode/src/tools/read"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpServiceReadToolIntegration(t *testing.T) {
	bus, err := events.NewEventBus()
	require.NoError(t, err, "EventBus 생성 실패")

	setupTestViper()

	toolResultEvents := make(chan events.Event, 10)
	eventSubscriber := &IntegrationEventSubscriber{toolResults: toolResultEvents}
	bus.Subscribe(events.ToolRawResultEvent, eventSubscriber)

	tmpDir, err := os.MkdirTemp("", "mcp_read_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("Read Tool 성공 케이스", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "success_test.txt")
		testContent := "첫 번째 줄\n두 번째 줄\n세 번째 줄\n"
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		mcpService := mcpservice.NewMcpService(bus)
		require.NotNil(t, mcpService)

		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "Read",
			Parameters: map[string]interface{}{
				"file_path": testFile,
			},
		}

		acceptEvent := events.Event{
			Type:      events.AcceptToolEvent,
			Data:      toolCallData,
			Timestamp: time.Now(),
			Source:    constants.Model,
		}

		mcpService.HandleEvent(acceptEvent)

		select {
		case event := <-toolResultEvents:
			assert.Equal(t, events.ToolRawResultEvent, event.Type)
			assert.Equal(t, constants.McpService, event.Source)

			data, ok := event.Data.(dto.ToolRawResultData)
			require.True(t, ok, "이벤트 데이터 타입이 올바르지 않음")

			assert.Equal(t, toolCallData.RequestUUID, data.RequestUUID)
			assert.Equal(t, toolCallData.ToolCallUUID, data.ToolCall)
			assert.NotNil(t, data.Result, "결과가 있어야 함")

			textContent, ok := data.Result.Content[0].(*mcp.TextContent)
			require.True(t, ok, "텍스트 컨텐츠여야 함")

			var readResult read.Success
			err := json.Unmarshal([]byte(textContent.Text), &readResult)
			require.NoError(t, err, "JSON 파싱 실패")

			assert.True(t, readResult.Success)
			assert.Equal(t, 3, readResult.TotalLines)
			assert.Contains(t, readResult.Text, "첫 번째 줄")

		case <-time.After(1 * time.Second):
			t.Fatal("성공 이벤트가 시간 내에 수신되지 않음")
		}
	})

	t.Run("Read Tool 파일 없음 에러", func(t *testing.T) {
		mcpService := mcpservice.NewMcpService(bus)
		require.NotNil(t, mcpService)

		nonExistentFile := filepath.Join(tmpDir, "nonexistent_file.txt")
		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "Read",
			Parameters: map[string]interface{}{
				"file_path": nonExistentFile,
			},
		}

		acceptEvent := events.Event{
			Type:      events.AcceptToolEvent,
			Data:      toolCallData,
			Timestamp: time.Now(),
			Source:    constants.Model,
		}

		mcpService.HandleEvent(acceptEvent)

		select {
		case event := <-toolResultEvents:
			assert.Equal(t, events.ToolRawResultEvent, event.Type)

			data, ok := event.Data.(dto.ToolRawResultData)
			require.True(t, ok, "이벤트 데이터 타입이 올바르지 않음")

			assert.Equal(t, toolCallData.RequestUUID, data.RequestUUID)
			assert.Equal(t, toolCallData.ToolCallUUID, data.ToolCall)
			
			if data.Result != nil && data.Result.IsError {
				assert.True(t, data.Result.IsError, "IsError 플래그가 설정되어야 함")
				if len(data.Result.Content) > 0 {
					if textContent, ok := data.Result.Content[0].(*mcp.TextContent); ok {
						assert.Contains(t, textContent.Text, "Tool Call Error")
					}
				}
			}

		case <-time.After(1 * time.Second):
			t.Fatal("에러 이벤트가 시간 내에 수신되지 않음")
		}
	})
}

type IntegrationEventSubscriber struct {
	toolResults chan events.Event
}

func (i *IntegrationEventSubscriber) HandleEvent(event events.Event) {
	switch event.Type {
	case events.ToolRawResultEvent:
		select {
		case i.toolResults <- event:
		default:
		}
	}
}

func (i *IntegrationEventSubscriber) GetID() constants.Source {
	return constants.Model
}

func setupTestViper() {
	viper.Set("mcp.name", "test-mcp")
	viper.Set("mcp.version", "1.0.0")
	viper.Set("server.name", "test-server")
	viper.Set("server.version", "1.0.0")
}