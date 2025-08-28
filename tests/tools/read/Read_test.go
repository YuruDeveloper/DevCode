package read_test

import (
	"DevCode/src/tools/read"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	desc := tool.Description()
	assert.Contains(t, desc, "Reads a file from the local filesystem")
	assert.Contains(t, desc, "file_path parameter must be an absolute")
}

func TestTool_Handler_ValidFile(t *testing.T) {
	// 임시 파일 생성
	tmpDir, err := os.MkdirTemp("", "read_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\n"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: testFile,
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	// 결과를 파싱하여 확인
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	var success read.Success
	err = json.Unmarshal([]byte(textContent.Text), &success)
	require.NoError(t, err)
	
	assert.True(t, success.Success)
	assert.Equal(t, 3, success.TotalLines)
	assert.Equal(t, 3, success.LinesRead)
	assert.Contains(t, success.Text, "     1\tLine 1")
	assert.Contains(t, success.Text, "     2\tLine 2")
	assert.Contains(t, success.Text, "     3\tLine 3")
}

func TestTool_Handler_FileNotFound(t *testing.T) {
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: "/nonexistent/path/file.txt",
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "file not found")
}

func TestTool_Handler_EmptyFilePath(t *testing.T) {
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

func TestTool_Handler_WithOffsetAndLimit(t *testing.T) {
	// 더 많은 줄이 있는 임시 파일 생성
	tmpDir, err := os.MkdirTemp("", "read_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	testFile := filepath.Join(tmpDir, "test_large.txt")
	testContent := ""
	for i := 1; i <= 10; i++ {
		testContent += fmt.Sprintf("Line %d\n", i)
	}
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: testFile,
			Offset:   3, // 3번째 줄부터
			Limit:    3, // 3줄만
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	var success read.Success
	err = json.Unmarshal([]byte(textContent.Text), &success)
	require.NoError(t, err)
	
	assert.True(t, success.Success)
	assert.Equal(t, 10, success.TotalLines)
	
	// Offset과 Limit이 적용되었는지 확인
	assert.Contains(t, success.Text, "     3\tLine 3")
	assert.Contains(t, success.Text, "     4\tLine 4")
	assert.Contains(t, success.Text, "     5\tLine 5")
	assert.NotContains(t, success.Text, "     1\tLine 1")
	assert.NotContains(t, success.Text, "     2\tLine 2")
}

func TestTool_Handler_EmptyFile(t *testing.T) {
	// 빈 파일 생성
	tmpDir, err := os.MkdirTemp("", "read_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	testFile := filepath.Join(tmpDir, "empty.txt")
	err = os.WriteFile(testFile, []byte(""), 0644)
	require.NoError(t, err)
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: testFile,
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	var success read.Success
	err = json.Unmarshal([]byte(textContent.Text), &success)
	require.NoError(t, err)
	
	assert.True(t, success.Success)
	assert.Equal(t, 0, success.TotalLines)
	assert.Equal(t, 0, success.LinesRead)
	assert.Equal(t, "", success.Text)
}

func TestTool_Handler_LongLines(t *testing.T) {
	// 긴 줄이 있는 파일 생성
	tmpDir, err := os.MkdirTemp("", "read_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	testFile := filepath.Join(tmpDir, "long_lines.txt")
	longLine := ""
	for i := 0; i < 100; i++ {
		longLine += "This is a very long line that should be handled properly. "
	}
	testContent := "Short line 1\n" + longLine + "\nShort line 2\n"
	
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: testFile,
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	var success read.Success
	err = json.Unmarshal([]byte(textContent.Text), &success)
	require.NoError(t, err)
	
	assert.True(t, success.Success)
	assert.Equal(t, 3, success.TotalLines)
	assert.Equal(t, 3, success.LinesRead)
	assert.Contains(t, success.Text, "     1\tShort line 1")
	assert.Contains(t, success.Text, "     2\t")  // 긴 줄의 시작
	assert.Contains(t, success.Text, "     3\tShort line 2")
}

func TestTool_Handler_MaxLinesLimit(t *testing.T) {
	// 2000줄 이상의 파일 생성
	tmpDir, err := os.MkdirTemp("", "read_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	testFile := filepath.Join(tmpDir, "large_file.txt")
	
	file, err := os.Create(testFile)
	require.NoError(t, err)
	
	// 2500줄 작성
	for i := 1; i <= 2500; i++ {
		_, err = file.WriteString(fmt.Sprintf("Line %d\n", i))
		require.NoError(t, err)
	}
	file.Close()
	
	tool := &read.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: testFile,
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	var success read.Success
	err = json.Unmarshal([]byte(textContent.Text), &success)
	require.NoError(t, err)
	
	assert.True(t, success.Success)
	// 최대 2000줄까지만 읽어야 함
	assert.Equal(t, 2000, success.TotalLines)
	assert.Equal(t, 2000, success.LinesRead)
	assert.Contains(t, success.Text, "  2000\tLine 2000")
	assert.NotContains(t, success.Text, "Line 2001")
}

func TestTool_Handler_PermissionDenied(t *testing.T) {
	// 권한이 거부된 파일에 대한 테스트
	tool := &read.Tool{}
	handler := tool.Handler()
	
	// /root 디렉터리는 일반 사용자가 접근할 수 없는 경우가 많음
	params := &mcp.CallToolParamsFor[read.Input]{
		Arguments: read.Input{
			FilePath: "/root/restricted_file.txt",
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	assert.Error(t, err)
	assert.Nil(t, result)
	// 권한 거부 또는 파일이 존재하지 않는 경우 모두 처리
	assert.True(t, 
		err.Error() == "permission denied: /root/restricted_file.txt" ||
		err.Error() == "file not found: /root/restricted_file.txt",
		"Expected permission denied or file not found error, got: %s", err.Error())
}

func TestTool_Handler_InvalidPathFormat(t *testing.T) {
	// 잘못된 경로 형식에 대한 테스트
	tool := &read.Tool{}
	handler := tool.Handler()
	
	testCases := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "Empty path",
			filePath: "",
			expected: "invalid path format",
		},
		{
			name:     "Relative path",
			filePath: "./relative/path.txt",
			expected: "file not found",
		},
		{
			name:     "Invalid characters",
			filePath: "/invalid\x00path/file.txt",
			expected: "invalid path format", // null character가 있으면 invalid path format 에러
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &mcp.CallToolParamsFor[read.Input]{
				Arguments: read.Input{
					FilePath: tc.filePath,
				},
			}
			
			result, err := handler(context.Background(), nil, params)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tc.expected)
		})
	}
}

func TestTool_Handler_ErrorMessages(t *testing.T) {
	// 다양한 에러 상황에서 적절한 에러 메시지가 반환되는지 확인
	tool := &read.Tool{}
	handler := tool.Handler()
	
	// Test Case 1: 존재하지 않는 파일
	t.Run("NonexistentFile", func(t *testing.T) {
		params := &mcp.CallToolParamsFor[read.Input]{
			Arguments: read.Input{
				FilePath: "/absolutely/nonexistent/file/path.txt",
			},
		}
		
		result, err := handler(context.Background(), nil, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "file not found: /absolutely/nonexistent/file/path.txt")
	})
	
	// Test Case 2: 빈 파일 경로
	t.Run("EmptyFilePath", func(t *testing.T) {
		params := &mcp.CallToolParamsFor[read.Input]{
			Arguments: read.Input{
				FilePath: "",
			},
		}
		
		result, err := handler(context.Background(), nil, params)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid path format:")
	})
	
	// Test Case 3: 디렉터리를 파일로 읽으려 시도
	t.Run("DirectoryAsFile", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "read_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)
		
		params := &mcp.CallToolParamsFor[read.Input]{
			Arguments: read.Input{
				FilePath: tmpDir, // 디렉터리 경로
			},
		}
		
		result, err := handler(context.Background(), nil, params)
		// 디렉터리를 읽으려 시도할 때의 동작을 확인
		// 실제로는 디렉터리를 읽으려고 시도하면 success로 처리되거나 에러가 발생할 수 있음
		if err != nil {
			assert.Contains(t, err.Error(), "invalid path format")
		} else {
			// 디렉터리를 성공적으로 "읽은" 경우 (빈 내용으로)
			require.NotNil(t, result)
			textContent, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok)
			
			var success read.Success
			err = json.Unmarshal([]byte(textContent.Text), &success)
			require.NoError(t, err)
			
			assert.True(t, success.Success)
			// 디렉터리는 내용이 없을 것으로 예상
			assert.Equal(t, "", success.Text)
		}
	})
}