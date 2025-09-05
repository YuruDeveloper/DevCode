package tools_test

import (
	"DevCode/src/tools"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTextReturn_Success(t *testing.T) {
	testText := "test data"
	
	result, err := tools.TextReturn(testText)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	// Content가 TextContent 타입인지 확인
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// 텍스트가 올바르게 설정되었는지 확인
	assert.Equal(t, testText, textContent.Text)
}

func TestTextReturn_WithComplexData(t *testing.T) {
	complexText := "복잡한 한국어 데이터와 특수문자 !@#$%^&*()\n\t줄바꿈과 탭 문자"
	
	result, err := tools.TextReturn(complexText)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// 복잡한 데이터가 올바르게 처리되었는지 확인
	assert.Equal(t, complexText, textContent.Text)
}

func TestTextReturn_EmptyString(t *testing.T) {
	emptyText := ""
	
	result, err := tools.TextReturn(emptyText)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	assert.Equal(t, "", textContent.Text)
}

func TestTextReturn_JSONString(t *testing.T) {
	jsonText := `{"test_data": "test value", "success": true}`
	
	result, err := tools.TextReturn(jsonText)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	assert.Equal(t, jsonText, textContent.Text)
}

func TestTextReturn_MultilineString(t *testing.T) {
	multilineText := `첫 번째 줄
두 번째 줄
세 번째 줄`
	
	result, err := tools.TextReturn(multilineText)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	assert.Equal(t, multilineText, textContent.Text)
}