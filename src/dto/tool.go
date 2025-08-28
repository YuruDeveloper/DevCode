package dto

import (
	"DevCode/src/constants"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RequestToolListData struct {
	CreateUUID uuid.UUID
}

type ToolListUpdateData struct {
	List []*mcp.Tool
}

type ToolCallData struct {
	RequestUUID  uuid.UUID
	ToolCallUUID uuid.UUID
	ToolName     string
	Parameters   map[string]any
}

type ToolResultData struct {
	RequestUUID uuid.UUID
	ToolCall    uuid.UUID
	ToolResult  string
}

type ToolRawResultData struct {
	RequestUUID uuid.UUID
	ToolCall    uuid.UUID
	Result      *mcp.CallToolResult
}

type ToolUseReportData struct {
	RequestUUID uuid.UUID
	ToolCall    uuid.UUID
	ToolInfo    string
	ToolStatus  constants.ToolStatus
}
