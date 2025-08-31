package ls_test

import (
	"DevCode/src/tools/ls"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_Name(t *testing.T) {
	tool := &ls.Tool{}
	assert.Equal(t, "LS", tool.Name())
}

func TestTool_Description(t *testing.T) {
	tool := &ls.Tool{}
	description := tool.Description()
	assert.Contains(t, description, "Lists files and directories")
	assert.Contains(t, description, "absolute path")
}

func TestTool_Handler_InvalidPath(t *testing.T) {
	tool := &ls.Tool{}
	handler := tool.Handler()
	
	// 상대 경로 테스트
	params := &mcp.CallToolParamsFor[ls.Input]{
		Arguments: ls.Input{
			Path: "relative/path",
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid Path")
}

func TestTool_Handler_EmptyPath(t *testing.T) {
	tool := &ls.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[ls.Input]{
		Arguments: ls.Input{
			Path: "",
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid Path")
}

func TestTool_Handler_NonExistentPath(t *testing.T) {
	tool := &ls.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[ls.Input]{
		Arguments: ls.Input{
			Path: "/non/existent/path",
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "directory not found")
}

func TestTool_Handler_ValidPath(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "ls_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// 테스트 파일과 디렉토리 생성
	testFile := filepath.Join(tempDir, "test_file.txt")
	testDir := filepath.Join(tempDir, "test_dir")
	
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	err = os.Mkdir(testDir, 0755)
	require.NoError(t, err)
	
	tool := &ls.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[ls.Input]{
		Arguments: ls.Input{
			Path: tempDir,
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// 결과가 예상한 형태인지 확인
	assert.Contains(t, textContent.Text, tempDir)
	assert.Contains(t, textContent.Text, "test_file.txt")
	assert.Contains(t, textContent.Text, "test_dir/")
}

func TestTool_Handler_WithIgnorePatterns(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "ls_test_ignore")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// 테스트 파일들 생성
	files := []string{
		"test1.txt",
		"test2.go",
		"ignore_me.tmp",
		"keep_me.md",
	}
	
	for _, fileName := range files {
		filePath := filepath.Join(tempDir, fileName)
		err = os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}
	
	tool := &ls.Tool{}
	handler := tool.Handler()
	
	params := &mcp.CallToolParamsFor[ls.Input]{
		Arguments: ls.Input{
			Path:   tempDir,
			Ignore: []string{"*.tmp"},
		},
	}
	
	result, err := handler(context.Background(), nil, params)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	
	// ignore_me.tmp는 결과에 포함되지 않아야 함
	assert.NotContains(t, textContent.Text, "ignore_me.tmp")
	// 다른 파일들은 포함되어야 함
	assert.Contains(t, textContent.Text, "test1.txt")
	assert.Contains(t, textContent.Text, "test2.go")
	assert.Contains(t, textContent.Text, "keep_me.md")
}

func TestTool_ShouldIgnore(t *testing.T) {
	tool := &ls.Tool{}
	
	testCases := []struct {
		name     string
		filename string
		patterns []string
		expected bool
	}{
		{
			name:     "단일 패턴 매칭",
			filename: "test.tmp",
			patterns: []string{"*.tmp"},
			expected: true,
		},
		{
			name:     "패턴 미매칭",
			filename: "test.txt",
			patterns: []string{"*.tmp"},
			expected: false,
		},
		{
			name:     "여러 패턴 중 하나 매칭",
			filename: "test.log",
			patterns: []string{"*.tmp", "*.log", "*.bak"},
			expected: true,
		},
		{
			name:     "빈 패턴 리스트",
			filename: "test.txt",
			patterns: []string{},
			expected: false,
		},
		{
			name:     "정확한 이름 매칭",
			filename: "ignore_me",
			patterns: []string{"ignore_me"},
			expected: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tool.ShouldIgnore(tc.filename, tc.patterns)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTool_Helper_NestedDirectories(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "ls_test_nested")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// 중첩된 디렉토리 구조 생성
	nestedDir := filepath.Join(tempDir, "level1", "level2")
	err = os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)
	
	// 각 레벨에 파일 생성
	files := map[string]string{
		filepath.Join(tempDir, "root_file.txt"):             "root content",
		filepath.Join(tempDir, "level1", "level1_file.txt"): "level1 content",
		filepath.Join(nestedDir, "level2_file.txt"):         "level2 content",
	}
	
	for filePath, content := range files {
		err = os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}
	
	tool := &ls.Tool{}
	result := tool.Helper(tempDir, []string{})
	
	// 결과에 모든 레벨의 파일과 디렉토리가 포함되는지 확인
	assert.Contains(t, result, "root_file.txt")
	assert.Contains(t, result, "level1/")
	assert.Contains(t, result, "level1_file.txt")
	assert.Contains(t, result, "level2/")
	assert.Contains(t, result, "level2_file.txt")
	
	// 들여쓰기가 제대로 되어있는지 확인 (간단한 검증)
	lines := strings.Split(result, "\n")
	var hasIndentedContent bool
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") {
			hasIndentedContent = true
			break
		}
	}
	assert.True(t, hasIndentedContent, "Should have indented content for nested structure")
}

func TestTool_Helper_EmptyDirectory(t *testing.T) {
	// 빈 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "ls_test_empty")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	tool := &ls.Tool{}
	result := tool.Helper(tempDir, []string{})
	
	// 빈 디렉토리는 빈 문자열을 반환해야 함
	assert.Equal(t, "", result)
}