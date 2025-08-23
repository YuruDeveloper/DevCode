package ollama

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ollama/ollama/api"
)

type Parameters struct {
	Type       string                      "json:\"type\""
	Defs       any                         "json:\"$defs,omitempty\""
	Items      any                         "json:\"items,omitempty\""
	Required   []string                    "json:\"required\""
	Properties map[string]api.ToolProperty "json:\"properties\""
}

func ConvertTool(mcpTool *mcp.Tool) api.Tool {
	ollamaTool := api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
		},
	}
	if mcpTool.InputSchema != nil {
		parameters := Parameters{
			Type:       "object",
			Required:   mcpTool.InputSchema.Required,
			Properties: make(map[string]api.ToolProperty),
		}
		for name, prop := range mcpTool.InputSchema.Properties {
			parameters.Properties[name] = api.ToolProperty{
				Type:        append(prop.Types, prop.Type),
				Description: prop.Description,
			}
		}
		ollamaTool.Function.Parameters = parameters
	}

	return ollamaTool
}
