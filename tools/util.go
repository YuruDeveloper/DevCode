package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TextReturn(input string) (*mcp.CallToolResultFor[any], error) {
	content := mcp.TextContent{Text: input}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&content,
		},
	}, nil
}
