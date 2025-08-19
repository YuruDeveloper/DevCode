package utils

import (
	"UniCode/src/types"
	"fmt"
	"strings"

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

func EnvironmentUpdateDataToString(data types.EnvironmentUpdateData) string {
	var builder strings.Builder
	builder.WriteString("<env>\n")
	builder.WriteString(fmt.Sprintf("Working directory: %s\n", data.Cwd))
	builder.WriteString(fmt.Sprintf("Is directory a git repo: %t\n", data.IsDirectoryGitRepo))
	builder.WriteString(fmt.Sprintf("Platform: %s\n", data.OS))
	builder.WriteString(fmt.Sprintf("OS Version: %s\n", data.OSVersion))
	builder.WriteString(fmt.Sprintf("Today's date: %s\n", data.TodayDate))
	builder.WriteString("</env>\n")
	return builder.String()
}
