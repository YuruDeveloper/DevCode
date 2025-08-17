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
	Handler() mcp.ToolHandlerFor[T,any]
}

type Result interface {
	Content() (*mcp.TextContent, error)
}

// 스타일 정의
const (
	BackgroundColor = lipgloss.Color("#0d1117")
	TextColor = lipgloss.Color("#f0f6fc")
	SubTextColor = lipgloss.Color("#8b949e")
	PrimaryColor   = lipgloss.Color("#1f6feb")
	SecondaryColor = lipgloss.Color("#2f81f7")
	AccentColor    = lipgloss.Color("#d29922")
	ErrorColor     = lipgloss.Color("#f85149")
	SuccessColor   = lipgloss.Color("#3fb950")
)

type Source int

const (
	McpService = iota + 1
	LLMService
	MessageService
	HistoryService
	EnvironmentService
	Model
)

//event data

type RequestData struct{
	SessionUUID uuid.UUID
	RequestUUID uuid.UUID 
	Message string
}

type EnviromentUpdateData struct {
	CreateUUID uuid.UUID 
	Cwd string
	OS string
	OSVersion string
	IsDirectoryGitRepo bool
	TodayDate string
}

type EnviromentRequestData struct {
	CreateUUID uuid.UUID
}

type RequestToolListData struct {
	CreateUUID uuid.UUID
}

type ToolListUpdateData struct {
	List []*mcp.Tool
}

type StreamChunk struct {
	Content string
	IsComplete bool
}

type StreamStartData struct {
	RequestUUID uuid.UUID
}

type StreamChunkData struct {
	RequestUUID uuid.UUID
	Chunk StreamChunk
}

type StreamCompleteData struct {
	RequestUUID uuid.UUID
	FinalMessage api.Message
	TotalChunks int
}

type SteramErrorData struct {
	RequestUUID uuid.UUID
	Error error
	ChunkCount int
}

type StreamCancelData struct {
	RequestUUID uuid.UUID
}

type ToolCallData struct {
	RequestUUID uuid.UUID
	ToolName string
	Paramters map[string]any
}


