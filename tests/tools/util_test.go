package tools_test

import (
	"DevCode/src/tools"
	"encoding/json"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Result implementation for testing
type MockResult struct {
	TestData string `json:"test_data"`
	Success  bool   `json:"success"`
}

func (m MockResult) Content() (*mcp.TextContent, error) {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return &mcp.TextContent{
		Text: string(jsonData),
	}, nil
}

// Mock Result with error for testing
type MockResultWithError struct {
}

func (m MockResultWithError) Content() (*mcp.TextContent, error) {
	// JSON marshaling이 실패할 수 있는 에러 반환
	return nil, errors.New("mock content error")
}

func TestTextReturn_Success(t *testing.T) {
	mockResult := MockResult{
		TestData: "test value",
		Success:  true,
	}
	
	result, err := tools.TextReturn(mockResult)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	// Content가 TextContent 타입인지 확인
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// JSON이 올바르게 마샬링되었는지 확인
	var unmarshaled MockResult
	err = json.Unmarshal([]byte(textContent.Text), &unmarshaled)
	require.NoError(t, err)
	
	assert.Equal(t, mockResult.TestData, unmarshaled.TestData)
	assert.Equal(t, mockResult.Success, unmarshaled.Success)
}

func TestTextReturn_WithComplexData(t *testing.T) {
	complexMockResult := MockResult{
		TestData: "복잡한 한국어 데이터와 특수문자 !@#$%^&*()",
		Success:  true,
	}
	
	result, err := tools.TextReturn(complexMockResult)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// 복잡한 데이터가 올바르게 처리되었는지 확인
	var unmarshaled MockResult
	err = json.Unmarshal([]byte(textContent.Text), &unmarshaled)
	require.NoError(t, err)
	
	assert.Equal(t, complexMockResult.TestData, unmarshaled.TestData)
	assert.Equal(t, complexMockResult.Success, unmarshaled.Success)
}

func TestTextReturn_EmptyResult(t *testing.T) {
	emptyMockResult := MockResult{
		TestData: "",
		Success:  false,
	}
	
	result, err := tools.TextReturn(emptyMockResult)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	var unmarshaled MockResult
	err = json.Unmarshal([]byte(textContent.Text), &unmarshaled)
	require.NoError(t, err)
	
	assert.Equal(t, "", unmarshaled.TestData)
	assert.Equal(t, false, unmarshaled.Success)
}

func TestTextReturn_ContentError(t *testing.T) {
	mockResultWithError := MockResultWithError{}
	
	result, err := tools.TextReturn(mockResultWithError)
	
	// Content() 메서드가 에러를 반환하면 TextReturn도 에러를 반환해야 함
	assert.Error(t, err)
	// result는 여전히 생성되어야 함 (에러와 함께 반환)
	assert.NotNil(t, result)
}

// 추가 테스트용 Result 구현
type ResultWithNilContent struct{}

func (r ResultWithNilContent) Content() (*mcp.TextContent, error) {
	return nil, nil
}

func TestTextReturn_NilContent(t *testing.T) {
	nilContentResult := ResultWithNilContent{}
	
	result, err := tools.TextReturn(nilContentResult)
	
	// 에러는 없지만 content가 nil
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	// Content[0]이 nil인지 확인
	assert.Nil(t, result.Content[0])
}