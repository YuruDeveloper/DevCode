package tool

import (
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"sync"
	"time"

	"go.uber.org/zap"
)

func NewToolManager(bus *events.EventBus, logger *zap.Logger) *ToolManager {
	manager := &ToolManager{
		bus:               bus,
		logger:            logger,
		activeTools:       make(map[types.ToolCallID]*types.ActiveTool),
		pendingToolStack:  make([]*types.PendingTool, 0, 5),
		changedActiveTool: make([]*types.ActiveTool, 0, 10),
	}
	manager.Subscribe()
	return manager
}

type ToolManager struct {
	bus               *events.EventBus
	logger            *zap.Logger
	activeTools       map[types.ToolCallID]*types.ActiveTool
	pendingToolStack  []*types.PendingTool
	changedActiveTool []*types.ActiveTool
	mutex             sync.Mutex
}

func (instance *ToolManager) Subscribe() {
	events.Subscribe(instance.bus, instance.bus.ToolUseReportEvent, constants.ToolManager, instance.ProcessReportEvent)
	events.Subscribe(instance.bus, instance.bus.RequestToolUseEvent, constants.ToolManager, instance.ProcessRequestEvent)
}

func (instance *ToolManager) ProcessRequestEvent(event events.Event[dto.ToolUseReportData]) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	instance.pendingToolStack = append(instance.pendingToolStack, &types.PendingTool{RequestID: event.Data.RequestID, ToolCallID: event.Data.ToolCallID})
	if len(instance.pendingToolStack) == 1 {
		events.Publish(instance.bus, instance.bus.UpdateUserStatusEvent, events.Event[dto.UpdateUserStatusData]{
			Data: dto.UpdateUserStatusData{
				Status: constants.ToolDecision,
			},
			TimeStamp: time.Now(),
			Source:    constants.ToolManager,
		})
	}
	// ProcessReportEvent를 직접 호출하지 않고 이벤트로 발행하여 데드락 방지
	events.Publish(instance.bus, instance.bus.ToolUseReportEvent, event)
}

func (instance *ToolManager) ProcessReportEvent(event events.Event[dto.ToolUseReportData]) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if activeTool, exists := instance.activeTools[event.Data.ToolCallID]; exists {
		if activeTool.ToolStatus != event.Data.ToolStatus || activeTool.ToolInfo != event.Data.ToolInfo {
			delete(instance.activeTools, event.Data.ToolCallID)
			instance.changedActiveTool = append(instance.changedActiveTool, &types.ActiveTool{
				ToolCallID: event.Data.ToolCallID,
				ToolStatus: event.Data.ToolStatus,
				ToolInfo:   event.Data.ToolInfo,
			})
			instance.PublishUpdateView()
		}
		return
	}
	activeTool := &types.ActiveTool{
		ToolStatus: event.Data.ToolStatus,
		ToolInfo:   event.Data.ToolInfo,
	}
	instance.activeTools[event.Data.ToolCallID] = activeTool
	instance.changedActiveTool = append(instance.changedActiveTool, activeTool)
	instance.PublishUpdateView()
}

func (instance *ToolManager) PublishUpdateView() {
	events.Publish(instance.bus, instance.bus.UpdateViewEvent, events.Event[dto.UpdateViewData]{
		Data:      dto.UpdateViewData{},
		TimeStamp: time.Now(),
		Source:    constants.ToolManager,
	})
}

func (instance *ToolManager) IsPending() bool {
	return len(instance.pendingToolStack) != 0
}

func (instance *ToolManager) ChangedActiveTool() []*types.ActiveTool {
	defer func() {
		instance.changedActiveTool = make([]*types.ActiveTool, 0)
	}()
	return instance.changedActiveTool
}

func (instance *ToolManager) Select(selectIndex int) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	accept := selectIndex == 0
	events.Publish(instance.bus, instance.bus.UserDecisionEvent,
		events.Event[dto.UserDecisionData]{
			Data: dto.UserDecisionData{
				RequestID:  instance.pendingToolStack[0].RequestID,
				ToolCallID: instance.pendingToolStack[0].ToolCallID,
				Accept:     accept,
			},
			TimeStamp: time.Now(),
			Source:    constants.ToolManager,
		})
	instance.pendingToolStack = instance.pendingToolStack[1:]
	instance.checkPeddingToolStack()
}

func (instance *ToolManager) Quit() {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	events.Publish(instance.bus, instance.bus.UserDecisionEvent,
		events.Event[dto.UserDecisionData]{
			Data: dto.UserDecisionData{
				RequestID:  instance.pendingToolStack[0].RequestID,
				ToolCallID: instance.pendingToolStack[0].ToolCallID,
				Accept:     false,
			},
			TimeStamp: time.Now(),
			Source:    constants.ToolManager,
		})
	instance.pendingToolStack = instance.pendingToolStack[1:]
	instance.checkPeddingToolStack()
}

func (instance *ToolManager) checkPeddingToolStack() {
	if len(instance.pendingToolStack) == 0 {
		events.Publish(instance.bus, instance.bus.UpdateUserStatusEvent, events.Event[dto.UpdateUserStatusData]{
			Data: dto.UpdateUserStatusData{
				Status: constants.AssistantInput,
			},
			TimeStamp: time.Now(),
			Source:    constants.ToolManager,
		})
	}
}
