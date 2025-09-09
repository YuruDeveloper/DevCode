package tool

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewToolManager(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	// When
	manager := NewToolManager(bus, logger)

	// Then
	assert.NotNil(t, manager)
	assert.Equal(t, bus, manager.bus)
	assert.Equal(t, logger, manager.logger)
	assert.NotNil(t, manager.activeTools)
	assert.NotNil(t, manager.pendingToolStack)
	assert.NotNil(t, manager.changedActiveTool)
	assert.Equal(t, 0, len(manager.activeTools))
	assert.Equal(t, 0, len(manager.pendingToolStack))
	assert.Equal(t, 0, len(manager.changedActiveTool))
}

func TestToolManager_ProcessRequestEvent(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()

	// When
	event := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  requestID,
			ToolCallID: toolCallID,
			ToolInfo:   "Test tool info",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}

	manager.ProcessRequestEvent(event)

	// Wait for async event processing
	time.Sleep(50 * time.Millisecond)

	// Then
	assert.True(t, manager.IsPending())
	assert.Equal(t, 1, len(manager.pendingToolStack))
	assert.Equal(t, requestID, manager.pendingToolStack[0].RequestID)
	assert.Equal(t, toolCallID, manager.pendingToolStack[0].ToolCallID)

	// Test that activeTool was created via the event
	assert.Equal(t, 1, len(manager.activeTools))
	activeTool, exists := manager.activeTools[toolCallID]
	assert.True(t, exists)
	assert.Equal(t, constants.Call, activeTool.ToolStatus)
	assert.Equal(t, "Test tool info", activeTool.ToolInfo)
}

func TestToolManager_ProcessReportEvent_NewTool(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)
	toolCallID := types.NewToolCallID()

	// When
	event := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  types.NewRequestID(),
			ToolCallID: toolCallID,
			ToolInfo:   "New tool info",
			ToolStatus: constants.Success,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}

	manager.ProcessReportEvent(event)

	// Then
	assert.Equal(t, 1, len(manager.activeTools))
	activeTool, exists := manager.activeTools[toolCallID]
	assert.True(t, exists)
	assert.Equal(t, constants.Success, activeTool.ToolStatus)
	assert.Equal(t, "New tool info", activeTool.ToolInfo)

	// Check changed active tools
	changedTools := manager.ChangedActiveTool()
	assert.Equal(t, 1, len(changedTools))
	assert.Equal(t, constants.Success, changedTools[0].ToolStatus)
	assert.Equal(t, "New tool info", changedTools[0].ToolInfo)

	// After getting changed tools, the slice should be reset
	changedTools2 := manager.ChangedActiveTool()
	assert.Equal(t, 0, len(changedTools2))
}

func TestToolManager_ProcessReportEvent_UpdateExistingTool(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)
	toolCallID := types.NewToolCallID()

	// Create initial tool
	initialEvent := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  types.NewRequestID(),
			ToolCallID: toolCallID,
			ToolInfo:   "Initial info",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}
	manager.ProcessReportEvent(initialEvent)

	// Clear changed tools
	manager.ChangedActiveTool()

	// When - Update with different status
	updateEvent := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  types.NewRequestID(),
			ToolCallID: toolCallID,
			ToolInfo:   "Updated info",
			ToolStatus: constants.Success,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}
	manager.ProcessReportEvent(updateEvent)

	// Then
	// Original tool should be removed from activeTools
	_, exists := manager.activeTools[toolCallID]
	assert.False(t, exists)

	// Check changed active tools has the updated tool
	changedTools := manager.ChangedActiveTool()
	assert.Equal(t, 1, len(changedTools))
	assert.Equal(t, constants.Success, changedTools[0].ToolStatus)
	assert.Equal(t, "Updated info", changedTools[0].ToolInfo)
}

func TestToolManager_ProcessReportEvent_NoChangeInExistingTool(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)
	toolCallID := types.NewToolCallID()

	// Create initial tool
	initialEvent := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  types.NewRequestID(),
			ToolCallID: toolCallID,
			ToolInfo:   "Same info",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}
	manager.ProcessReportEvent(initialEvent)

	// Clear changed tools
	manager.ChangedActiveTool()

	// When - Send same status and info
	sameEvent := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  types.NewRequestID(),
			ToolCallID: toolCallID,
			ToolInfo:   "Same info",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}
	manager.ProcessReportEvent(sameEvent)

	// Then
	// Tool should remain in activeTools
	activeTool, exists := manager.activeTools[toolCallID]
	assert.True(t, exists)
	assert.Equal(t, constants.Call, activeTool.ToolStatus)

	// No changes should be recorded
	changedTools := manager.ChangedActiveTool()
	assert.Equal(t, 0, len(changedTools))
}

func TestToolManager_IsPending(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)

	// When - Initially no pending tools
	// Then
	assert.False(t, manager.IsPending())

	// When - Add pending tool
	event := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  types.NewRequestID(),
			ToolCallID: types.NewToolCallID(),
			ToolInfo:   "Test",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}
	manager.ProcessRequestEvent(event)

	// Then
	assert.True(t, manager.IsPending())
}

func TestToolManager_Select(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()

	// Add a pending tool
	event := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  requestID,
			ToolCallID: toolCallID,
			ToolInfo:   "Test",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}
	manager.ProcessRequestEvent(event)

	// When - Select accept (index 0)
	manager.Select(0)

	// Then
	assert.False(t, manager.IsPending())
}

func TestToolManager_Select_Reject(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()

	// Add a pending tool
	event := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  requestID,
			ToolCallID: toolCallID,
			ToolInfo:   "Test",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}
	manager.ProcessRequestEvent(event)

	// When - Select reject (index 1)
	manager.Select(1)

	// Then
	assert.False(t, manager.IsPending())
}

func TestToolManager_Quit(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()

	// Add a pending tool
	event := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  requestID,
			ToolCallID: toolCallID,
			ToolInfo:   "Test",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}
	manager.ProcessRequestEvent(event)

	// When
	manager.Quit()

	// Then
	assert.False(t, manager.IsPending())
}

func TestToolManager_MultiplePendingTools(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)

	// Add multiple pending tools
	requestID1 := types.NewRequestID()
	toolCallID1 := types.NewToolCallID()
	requestID2 := types.NewRequestID()
	toolCallID2 := types.NewToolCallID()

	event1 := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  requestID1,
			ToolCallID: toolCallID1,
			ToolInfo:   "Test 1",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}

	event2 := events.Event[dto.ToolUseReportData]{
		Data: dto.ToolUseReportData{
			RequestID:  requestID2,
			ToolCallID: toolCallID2,
			ToolInfo:   "Test 2",
			ToolStatus: constants.Call,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}

	manager.ProcessRequestEvent(event1)
	manager.ProcessRequestEvent(event2)

	// Then
	assert.True(t, manager.IsPending())
	assert.Equal(t, 2, len(manager.pendingToolStack))

	// When - Handle first tool
	manager.Select(0) // Accept first tool

	// Then - Should still be pending for second tool
	assert.True(t, manager.IsPending())
	assert.Equal(t, 1, len(manager.pendingToolStack))
	assert.Equal(t, requestID2, manager.pendingToolStack[0].RequestID)

	// When - Handle second tool
	manager.Select(1) // Reject second tool

	// Then - Should not be pending anymore
	assert.False(t, manager.IsPending())
	assert.Equal(t, 0, len(manager.pendingToolStack))
}

func TestToolManager_ConcurrentAccess(t *testing.T) {
	// Given
	logger := zap.NewNop()
	bus, err := events.NewEventBus(config.EventBusConfig{PoolSize: 100}, logger)
	require.NoError(t, err)
	defer bus.Close()

	manager := NewToolManager(bus, logger)

	// When - Simulate concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Add pending tools
	go func() {
		for i := 0; i < 10; i++ {
			event := events.Event[dto.ToolUseReportData]{
				Data: dto.ToolUseReportData{
					RequestID:  types.NewRequestID(),
					ToolCallID: types.NewToolCallID(),
					ToolInfo:   "Concurrent test",
					ToolStatus: constants.Call,
				},
				TimeStamp: time.Now(),
				Source:    constants.ToolModule,
			}
			manager.ProcessRequestEvent(event)
		}
		done <- true
	}()

	// Goroutine 2: Process reports
	go func() {
		for i := 0; i < 10; i++ {
			event := events.Event[dto.ToolUseReportData]{
				Data: dto.ToolUseReportData{
					RequestID:  types.NewRequestID(),
					ToolCallID: types.NewToolCallID(),
					ToolInfo:   "Report test",
					ToolStatus: constants.Success,
				},
				TimeStamp: time.Now(),
				Source:    constants.ToolModule,
			}
			manager.ProcessReportEvent(event)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Then - Should not panic and should have some pending tools
	assert.True(t, manager.IsPending())
	assert.True(t, len(manager.pendingToolStack) > 0)
	assert.True(t, len(manager.activeTools) >= 10) // At least from reports
}
