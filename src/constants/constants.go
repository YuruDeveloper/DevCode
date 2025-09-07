package constants

import "fmt"

type Source int

const (
	McpService = Source(iota + 1)
	LLMService
	MessageService
	HistoryService
	EnvironmentService
	ToolService
	Model
	ToolManager
)

func (instance Source) String() string {
	switch instance {
	case McpService:
		return "McpService"
	case LLMService:
		return "LLMService"
	case MessageService:
		return "MessageService"
	case HistoryService:
		return "HistoryService"
	case EnvironmentService:
		return "EnvironmentService"
	case ToolService:
		return "ToolService"
	case Model:
		return "Model"
	default:
		return fmt.Sprintf("Source(%d)", int(instance))
	}
}

type ToolStatus int

const (
	Call = ToolStatus(iota + 1)
	Success
	Error
)

type UserStatus int

const (
	UserInput = UserStatus(iota + 1)
	AssistantInput
	ToolDecision
)
