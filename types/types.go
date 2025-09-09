package types

import (
	"DevCode/constants"
)

type ToolManager interface {
	IsPending() bool
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
