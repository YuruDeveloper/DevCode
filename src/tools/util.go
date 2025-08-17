package tools

import (
	"UniCode/src/types"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)


func TextReturn(input types.Result) (*mcp.CallToolResultFor[any] ,error){
	json , err := input.Content()
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content {
			json ,
		},
	} , err
}