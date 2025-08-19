package types

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ollama/ollama/api"
)

type Tool[T any] interface {
	Name() string
	Description() string
	Handler() mcp.ToolHandlerFor[T, any]
}

type Result interface {
	Content() (*mcp.TextContent, error)
}

// 스타일 정의
const (
	BackgroundColor = lipgloss.Color("#0d1117")
	TextColor       = lipgloss.Color("#f0f6fc")
	SubTextColor    = lipgloss.Color("#8b949e")
	PrimaryColor    = lipgloss.Color("#1f6feb")
	SecondaryColor  = lipgloss.Color("#2f81f7")
	AccentColor     = lipgloss.Color("#d29922")
	ErrorColor      = lipgloss.Color("#f85149")
	SuccessColor    = lipgloss.Color("#3fb950")
)

type Source int

const (
	McpService = iota + 1
	LLMService
	MessageService
	HistoryService
	EnvironmentService
	ToolService
	Model
)

type ToolStauts int

const (
	Call = iota + 100
	Success
	Error
)

//event data

type RequestData struct {
	SessionUUID uuid.UUID
	RequestUUID uuid.UUID
	Message     string
}

type EnvironmentUpdateData struct {
	CreateUUID         uuid.UUID
	Cwd                string
	OS                 string
	OSVersion          string
	IsDirectoryGitRepo bool
	TodayDate          string
}

type EnvironmentRequestData struct {
	CreateUUID uuid.UUID
}

type RequestToolListData struct {
	CreateUUID uuid.UUID
}

type ToolListUpdateData struct {
	List []*mcp.Tool
}

type StreamStartData struct {
	RequestUUID uuid.UUID
}

type StreamChunkData struct {
	RequestUUID uuid.UUID
	Content     string
	IsComplete  bool
}

type StreamCompleteData struct {
	RequestUUID  uuid.UUID
	FinalMessage api.Message
	IsComplete   bool
}

type StreamErrorData struct {
	RequestUUID uuid.UUID
	Error       error
	ChunkCount  int
}

type StreamCancelData struct {
	RequestUUID uuid.UUID
}

type ToolCallData struct {
	RequestUUID uuid.UUID
	ToolCall    uuid.UUID
	ToolName    string
	Parameters  map[string]any
}

type ParsedChunkData struct {
	RequestUUID uuid.UUID
	Content     string
	IsComplete  bool
}

type ParsedChunkErrorData struct {
	RequestUUID uuid.UUID
	Error       string
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
	Error       error
}

type ToolUseReportData struct {
	RequestUUID uuid.UUID
	ToolCall    uuid.UUID
	ToolInfo    string
	ToolStatus  ToolStauts
}

type UserDecisionData struct {
	RequestUUID uuid.UUID
	ToolCall    uuid.UUID
	Aceept      bool
}
