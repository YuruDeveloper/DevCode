package types

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Tool[T any] interface {
	Name() string
	Description() string
	Handler() mcp.ToolHandlerFor[T, any]
}
