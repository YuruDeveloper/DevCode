package ollama

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"context"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewStreamManager(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultActiveStreamSize: 5,
	}

	manager := NewStreamManager(ollamaConfig)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.ctxs)
	assert.NotNil(t, manager.activeStreams)
	assert.Equal(t, "", manager.buffer)
	assert.Equal(t, 0, len(manager.ctxs)) // 빈 맵으로 시작
	assert.Equal(t, 0, len(manager.activeStreams))
}

func TestStreamManager_Response_WithContent(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultActiveStreamSize: 5,
	}
	manager := NewStreamManager(ollamaConfig)

	logger := zap.NewNop()
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	requestID := types.NewRequestID()
	response := api.ChatResponse{
		Message: api.Message{
			Content: "Test content",
		},
		Done: false,
	}

	// Create channel to capture stream chunk event
	chunkEventsChan := make(chan events.Event[dto.StreamChunkData], 1)
	events.Subscribe(bus, bus.StreamChunkEvent, TestModule, func(event events.Event[dto.StreamChunkData]) {
		chunkEventsChan <- event
	})

	doneCallbackCalled := false
	doneCallback := func(message string) {
		doneCallbackCalled = true
	}

	checkDone := func(requestID types.RequestID) bool {
		return false
	}

	toolsCallback := func(requestID types.RequestID, toolCalls []api.ToolCall) {
		// Not expected for this test
	}

	err = manager.Response(requestID, response, bus, doneCallback, checkDone, toolsCallback)

	require.NoError(t, err)

	// Verify stream chunk event was published
	select {
	case event := <-chunkEventsChan:
		assert.Equal(t, requestID, event.Data.RequestID)
		assert.Equal(t, "Test content", event.Data.Content)
		assert.False(t, event.Data.IsComplete)
		assert.Equal(t, constants.LLMModule, event.Source)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected StreamChunkEvent was not published")
	}

	// Buffer should contain the content
	assert.Equal(t, "Test content", manager.buffer)

	// Done callback should not be called for incomplete response
	assert.False(t, doneCallbackCalled)
}

func TestStreamManager_Response_Done(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultActiveStreamSize: 5,
	}
	manager := NewStreamManager(ollamaConfig)
	manager.buffer = "Previous content"

	logger := zap.NewNop()
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	requestID := types.NewRequestID()
	response := api.ChatResponse{
		Message: api.Message{
			Content: "Final content",
		},
		Done: true,
	}

	// Create channels to capture events
	chunkEventsChan := make(chan events.Event[dto.StreamChunkData], 1)
	completeEventsChan := make(chan events.Event[dto.StreamCompleteData], 1)

	events.Subscribe(bus, bus.StreamChunkEvent, TestModule, func(event events.Event[dto.StreamChunkData]) {
		chunkEventsChan <- event
	})

	events.Subscribe(bus, bus.StreamCompleteEvent, TestModule, func(event events.Event[dto.StreamCompleteData]) {
		completeEventsChan <- event
	})

	doneCallbackMessage := ""
	doneCallback := func(message string) {
		doneCallbackMessage = message
	}

	checkDone := func(requestID types.RequestID) bool {
		return false // No pending calls
	}

	toolsCallback := func(requestID types.RequestID, toolCalls []api.ToolCall) {
		// Not expected for this test
	}

	err = manager.Response(requestID, response, bus, doneCallback, checkDone, toolsCallback)

	require.NoError(t, err)

	// Verify stream chunk event
	select {
	case event := <-chunkEventsChan:
		assert.Equal(t, requestID, event.Data.RequestID)
		assert.Equal(t, "Final content", event.Data.Content)
		assert.True(t, event.Data.IsComplete)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected StreamChunkEvent was not published")
	}

	// Verify stream complete event
	select {
	case event := <-completeEventsChan:
		assert.Equal(t, requestID, event.Data.RequestID)
		assert.Equal(t, "Final content", event.Data.FinalMessage)
		assert.True(t, event.Data.IsComplete)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected StreamCompleteEvent was not published")
	}

	// Done callback should be called with buffered content
	assert.Equal(t, "Previous contentFinal content", doneCallbackMessage)

	// Buffer should be cleared
	assert.Equal(t, "", manager.buffer)
}

func TestStreamManager_Response_WithToolCalls(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultActiveStreamSize: 5,
	}
	manager := NewStreamManager(ollamaConfig)

	logger := zap.NewNop()
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	requestID := types.NewRequestID()
	toolCalls := []api.ToolCall{
		{
			Function: api.ToolCallFunction{
				Name:      "test-tool",
				Arguments: map[string]interface{}{"param": "value"},
			},
		},
	}

	response := api.ChatResponse{
		Message: api.Message{
			Content:   "Response with tool calls",
			ToolCalls: toolCalls,
		},
		Done: true,
	}

	toolsCallbackCalled := false
	var capturedToolCalls []api.ToolCall
	var capturedRequestID types.RequestID

	doneCallback := func(message string) {}
	checkDone := func(requestID types.RequestID) bool { return false }
	toolsCallback := func(requestID types.RequestID, calls []api.ToolCall) {
		toolsCallbackCalled = true
		capturedRequestID = requestID
		capturedToolCalls = calls
	}

	err = manager.Response(requestID, response, bus, doneCallback, checkDone, toolsCallback)

	require.NoError(t, err)
	assert.True(t, toolsCallbackCalled)
	assert.Equal(t, requestID, capturedRequestID)
	assert.Equal(t, toolCalls, capturedToolCalls)
}

func TestStreamManager_CancelStream(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultActiveStreamSize: 5,
	}
	manager := NewStreamManager(ollamaConfig)

	requestID := types.NewRequestID()
	ctx, cancel := context.WithCancel(context.Background())

	// Simulate active stream
	manager.ctxs[requestID] = ctx
	manager.activeStreams[requestID] = cancel

	// Verify stream is active
	assert.Contains(t, manager.ctxs, requestID)
	assert.Contains(t, manager.activeStreams, requestID)

	// Cancel stream
	manager.CancelStream(requestID)

	// Verify stream is removed
	assert.NotContains(t, manager.ctxs, requestID)
	assert.NotContains(t, manager.activeStreams, requestID)
}

func TestStreamManager_CancelStream_NonExistentStream(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultActiveStreamSize: 5,
	}
	manager := NewStreamManager(ollamaConfig)

	requestID := types.NewRequestID()

	// Cancel non-existent stream should not panic
	manager.CancelStream(requestID)

	// Verify no streams exist
	assert.Empty(t, manager.ctxs)
	assert.Empty(t, manager.activeStreams)
}

func TestStreamManager_Response_NoContent(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultActiveStreamSize: 5,
	}
	manager := NewStreamManager(ollamaConfig)

	logger := zap.NewNop()
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	requestID := types.NewRequestID()
	response := api.ChatResponse{
		Message: api.Message{
			Content: "", // Empty content
		},
		Done: false,
	}

	// Create channel to capture stream events
	eventsChan := make(chan events.Event[dto.StreamChunkData], 1)
	events.Subscribe(bus, bus.StreamChunkEvent, TestModule, func(event events.Event[dto.StreamChunkData]) {
		eventsChan <- event
	})

	doneCallback := func(message string) {}
	checkDone := func(requestID types.RequestID) bool { return false }
	toolsCallback := func(requestID types.RequestID, toolCalls []api.ToolCall) {}

	err = manager.Response(requestID, response, bus, doneCallback, checkDone, toolsCallback)

	require.NoError(t, err)

	// No event should be published for empty content
	select {
	case <-eventsChan:
		t.Fatal("No event should be published for empty content")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event published
	}

	// Buffer should remain empty
	assert.Equal(t, "", manager.buffer)
}
