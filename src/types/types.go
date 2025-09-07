package types

import (
	"DevCode/src/constants"
)

type ToolManager interface {
	IsPedding() bool
	ChangedActiveTool() []*ActiveTool
	Select(selectIndex int)
	Quit()
}

type ActiveTool struct {
	ToolCallID ToolCallID
	ToolInfo   string
	ToolStatus constants.ToolStatus
}

type PendingTool struct {
	RequestID  RequestID
	ToolCallID ToolCallID
}
