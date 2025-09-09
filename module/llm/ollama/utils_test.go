package ollama

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
)

func TestConvertTool_BasicTool(t *testing.T) {
	mcpTool := &mcp.Tool{
		Name:        "test-tool",
		Description: "A test tool for unit testing",
	}

	ollamaTool := ConvertTool(mcpTool)

	assert.Equal(t, "function", ollamaTool.Type)
	assert.Equal(t, "test-tool", ollamaTool.Function.Name)
	assert.Equal(t, "A test tool for unit testing", ollamaTool.Function.Description)
	// InputSchema가 nil이어도 Parameters는 빈 구조체가 될 수 있음
}

func TestConvertTool_EmptyName(t *testing.T) {
	mcpTool := &mcp.Tool{
		Name:        "",
		Description: "Tool with empty name",
	}

	ollamaTool := ConvertTool(mcpTool)

	assert.Equal(t, "function", ollamaTool.Type)
	assert.Equal(t, "", ollamaTool.Function.Name)
	assert.Equal(t, "Tool with empty name", ollamaTool.Function.Description)
}

func TestConvertTool_EmptyDescription(t *testing.T) {
	mcpTool := &mcp.Tool{
		Name:        "unnamed-tool",
		Description: "",
	}

	ollamaTool := ConvertTool(mcpTool)

	assert.Equal(t, "function", ollamaTool.Type)
	assert.Equal(t, "unnamed-tool", ollamaTool.Function.Name)
	assert.Equal(t, "", ollamaTool.Function.Description)
}

func TestParameters_Structure(t *testing.T) {
	// Test the Parameters struct directly
	params := Parameters{
		Type:     "object",
		Required: []string{"param1", "param2"},
		Properties: map[string]api.ToolProperty{
			"param1": {
				Type:        []string{"string"},
				Description: "Parameter 1",
			},
			"param2": {
				Type:        []string{"number"},
				Description: "Parameter 2",
			},
		},
	}

	assert.Equal(t, "object", params.Type)
	assert.Equal(t, 2, len(params.Required))
	assert.Equal(t, 2, len(params.Properties))
	assert.Contains(t, params.Required, "param1")
	assert.Contains(t, params.Required, "param2")
}

func TestParameters_EmptyStructure(t *testing.T) {
	params := Parameters{
		Type:       "object",
		Required:   []string{},
		Properties: map[string]api.ToolProperty{},
	}

	assert.Equal(t, "object", params.Type)
	assert.Equal(t, 0, len(params.Required))
	assert.Equal(t, 0, len(params.Properties))
}

func TestConvertTool_AlwaysReturnsFunctionType(t *testing.T) {
	testCases := []*mcp.Tool{
		{Name: "tool1", Description: "desc1"},
		{Name: "tool2", Description: "desc2"},
		{Name: "", Description: ""},
	}

	for _, mcpTool := range testCases {
		ollamaTool := ConvertTool(mcpTool)
		assert.Equal(t, "function", ollamaTool.Type, "All tools should have type 'function'")
		assert.Equal(t, mcpTool.Name, ollamaTool.Function.Name)
		assert.Equal(t, mcpTool.Description, ollamaTool.Function.Description)
	}
}
