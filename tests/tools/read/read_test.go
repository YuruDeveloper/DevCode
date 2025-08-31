package read_test

import (
	"DevCode/src/tools/read"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_Name(t *testing.T) {
	tool := &read.Tool{}
	assert.Equal(t, "Read", tool.Name())
}

func TestTool_Description(t *testing.T) {
	tool := &read.Tool{}
	description := tool.Description()
	assert.Contains(t, description, "Reads a file from the local filesystem")
	assert.Contains(t, description, "file_path parameter must be an absolute")
}

func TestTool_Handler_InvalidPath(t *testing.T) {
	tool := &read.Tool{}
	handler := tool.Handler()
	
	// 상대 경로 테스트
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: "relative/path/file.txt",
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid path format")
}

func TestTool_Handler_EmptyPath(t *testing.T) {
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: "",
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid path format")
}

func TestTool_Handler_NonExistentFile(t *testing.T) {
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: "/non/existent/file.txt",
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "file not found")
}

func TestTool_Handler_ValidFile(t *testing.T) {
	// 임시 파일 생성
	tempFile, err := os.CreateTemp("", "read_test_*.txt")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	
	// 테스트 내용 작성
	testContent := "Line 1\nLine 2\nLine 3\n한국어 테스트\n특수문자 !@#$%^&*()"
	_, err = tempFile.WriteString(testContent)
	require.NoError(t, err)
	tempFile.Close()
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: tempFile.Name(),
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// 결과가 올바른 형태인지 확인 (line number → content 형식)
	assert.Contains(t, textContent.Text, "1→\tLine 1")
	assert.Contains(t, textContent.Text, "2→\tLine 2")
	assert.Contains(t, textContent.Text, "3→\tLine 3")
	assert.Contains(t, textContent.Text, "4→\t한국어 테스트")
	assert.Contains(t, textContent.Text, "5→\t특수문자 !@#$%^&*()")
}

func TestTool_Handler_WithOffset(t *testing.T) {
	// 임시 파일 생성
	tempFile, err := os.CreateTemp("", "read_test_offset_*.txt")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	
	// 여러 줄의 테스트 내용 작성
	lines := []string{
		"Line 1",
		"Line 2", 
		"Line 3",
		"Line 4",
		"Line 5",
	}
	testContent := strings.Join(lines, "\n")
	_, err = tempFile.WriteString(testContent)
	require.NoError(t, err)
	tempFile.Close()
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	// Offset 3부터 읽기 (Line 3부터)
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: tempFile.Name(),
			Offset:   3,
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// Offset 이전 줄들은 포함되지 않아야 함
	assert.NotContains(t, textContent.Text, "1→\tLine 1")
	assert.NotContains(t, textContent.Text, "2→\tLine 2")
	// Offset 이후 줄들은 포함되어야 함
	assert.Contains(t, textContent.Text, "3→\tLine 3")
	assert.Contains(t, textContent.Text, "4→\tLine 4")
	assert.Contains(t, textContent.Text, "5→\tLine 5")
}

func TestTool_Handler_WithLimit(t *testing.T) {
	// 임시 파일 생성
	tempFile, err := os.CreateTemp("", "read_test_limit_*.txt")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	
	// 여러 줄의 테스트 내용 작성
	lines := []string{
		"Line 1",
		"Line 2", 
		"Line 3",
		"Line 4",
		"Line 5",
	}
	testContent := strings.Join(lines, "\n")
	_, err = tempFile.WriteString(testContent)
	require.NoError(t, err)
	tempFile.Close()
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	// 최대 2줄만 읽기
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: tempFile.Name(),
			Limit:    2,
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// 처음 2줄만 포함되어야 함
	assert.Contains(t, textContent.Text, "1→\tLine 1")
	assert.Contains(t, textContent.Text, "2→\tLine 2")
	// 나머지 줄들은 포함되지 않아야 함
	assert.NotContains(t, textContent.Text, "3→\tLine 3")
	assert.NotContains(t, textContent.Text, "4→\tLine 4")
	assert.NotContains(t, textContent.Text, "5→\tLine 5")
}

func TestTool_Handler_WithOffsetAndLimit(t *testing.T) {
	// 임시 파일 생성
	tempFile, err := os.CreateTemp("", "read_test_offset_limit_*.txt")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	
	// 여러 줄의 테스트 내용 작성
	lines := []string{
		"Line 1",
		"Line 2", 
		"Line 3",
		"Line 4",
		"Line 5",
		"Line 6",
	}
	testContent := strings.Join(lines, "\n")
	_, err = tempFile.WriteString(testContent)
	require.NoError(t, err)
	tempFile.Close()
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	// Offset 2부터 2줄만 읽기 (Line 2, Line 3)
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: tempFile.Name(),
			Offset:   2,
			Limit:    2,
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// 첫 번째 줄은 건너뛰어야 함
	assert.NotContains(t, textContent.Text, "1→\tLine 1")
	// Line 2, Line 3만 포함되어야 함
	assert.Contains(t, textContent.Text, "2→\tLine 2")
	assert.Contains(t, textContent.Text, "3→\tLine 3")
	// 나머지 줄들은 포함되지 않아야 함
	assert.NotContains(t, textContent.Text, "4→\tLine 4")
	assert.NotContains(t, textContent.Text, "5→\tLine 5")
	assert.NotContains(t, textContent.Text, "6→\tLine 6")
}

func TestTool_Handler_EmptyFile(t *testing.T) {
	// 빈 임시 파일 생성
	tempFile, err := os.CreateTemp("", "read_test_empty_*.txt")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: tempFile.Name(),
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// 빈 파일은 빈 내용을 반환해야 함
	assert.Equal(t, "", textContent.Text)
}

func TestTool_Handler_LargeFile_2000LineLimit(t *testing.T) {
	// 임시 파일 생성
	tempFile, err := os.CreateTemp("", "read_test_large_*.txt")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	
	// 2500줄의 큰 파일 생성 (2000줄 제한 테스트)
	var lines []string
	for i := 1; i <= 2500; i++ {
		lines = append(lines, "This is line "+string(rune(i+48))) // 간단한 내용
	}
	testContent := strings.Join(lines, "\n")
	_, err = tempFile.WriteString(testContent)
	require.NoError(t, err)
	tempFile.Close()
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: tempFile.Name(),
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// 2000줄까지만 읽혀야 함
	lineCount := strings.Count(textContent.Text, "\n")
	assert.LessOrEqual(t, lineCount, 2000)
	
	// 처음 몇 줄과 2000번째 줄은 포함되어야 함
	assert.Contains(t, textContent.Text, "1→\t")
	assert.Contains(t, textContent.Text, "2000→\t")
	// 2001번째 줄 이후는 포함되지 않아야 함 (2000줄 제한)
	assert.NotContains(t, textContent.Text, "2001→\t")
}

func TestTool_Handler_PermissionDenied(t *testing.T) {
	// 이 테스트는 실제 권한이 없는 파일에 대해서만 의미가 있음
	// 시스템에 따라 다를 수 있으므로 skip할 수도 있음
	tool := &read.Tool{}
	handler := tool.Handler()
	
	// 일반적으로 권한이 없는 시스템 파일 (예: /etc/shadow)
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: "/etc/shadow", // 일반적으로 읽기 권한이 없는 파일
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	if err != nil {
		// 권한 에러가 발생하는 경우
		assert.Contains(t, err.Error(), "permission denied")
		assert.Nil(t, result)
	} else {
		// 파일이 존재하지 않거나 권한이 있는 경우 - 이는 시스템에 따라 다름
		t.Skip("Permission test skipped - file may not exist or permission may be available")
	}
}