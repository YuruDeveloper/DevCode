package constants

import "fmt"

type Source int

const (
	McpModule = Source(iota + 1)
	LLMModule
	MessageModule
	HistoryModule
	EnvironmentModule
	ToolModule
	Model
	ToolManager
)

func (instance Source) String() string {
	switch instance {
	case McpModule:
		return "McpModule"
	case LLMModule:
		return "LLMModule"
	case MessageModule:
		return "MessageModule"
	case HistoryModule:
		return "HistoryModule"
	case EnvironmentModule:
		return "EnvironmentModule"
	case ToolModule:
		return "ToolModule"
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
