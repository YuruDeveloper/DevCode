package dto

import (
	"DevCode/src/constants"
	"DevCode/src/types"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RequestToolListData struct {
	CreateID types.CreateID
}

type ToolListUpdateData struct {
	List []*mcp.Tool
}

type ToolCallData struct {
	RequestID  types.RequestID
	ToolCallID types.ToolCallID
	ToolName   string
	Parameters map[string]any
}

type ToolResultData struct {
	RequestID  types.RequestID
	ToolCallID types.ToolCallID
	ToolResult string
}

type ToolRawResultData struct {
	RequestID  types.RequestID
	ToolCallID types.ToolCallID
	Result     *mcp.CallToolResult
}

type ToolUseReportData struct {
	RequestID  types.RequestID
	ToolCallID types.ToolCallID
	ToolInfo   string
	ToolStatus constants.ToolStatus
}
