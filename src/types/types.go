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

type ToolListUpdate struct {
	List []*mcp.Tool
}

type ResponseData struct {
	Message api.Message
}