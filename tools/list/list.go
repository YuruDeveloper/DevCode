package list

import (
	"DevCode/tools"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	LsDescription = `Lists files and directories in a given path. The path parameter
  must be an absolute path, not a relative path. You can optionally provide an array
  of glob patterns to ignore with the ignore parameter. You should generally prefer
  the Glob and Grep tools, if you know which directories to search.`
	Name = "List"
)

type Tool struct {
}

func (*Tool) Name() string {
	return Name
}

func (*Tool) Description() string {
	return LsDescription
}

func (instance *Tool) Handler() mcp.ToolHandlerFor[Input, any] {
	return func(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[Input]) (*mcp.CallToolResultFor[any], error) {
		input := params.Arguments
		if input.Path == "" || !filepath.IsAbs(input.Path) {
			return nil, fmt.Errorf("invalid Path : %s", input.Path)
		}
		if _, err := os.Stat(input.Path); os.IsNotExist(err) {
			return nil, fmt.Errorf("directory not found: %s", input.Path)
		}
		_, err := os.ReadDir(input.Path)
		if err != nil {
			if os.IsPermission(err) {
				return nil, fmt.Errorf("permission denied : %s", input.Path)
			}
			return nil, fmt.Errorf("invalid path : %s", input.Path)
		}
		result := input.Path + "/\n" + instance.Helper(input.Path, input.Ignore)
		return tools.TextReturn(result)
	}
}

func (instance *Tool) Helper(dir string, ignore []string) string {
	visited := make(map[string]bool, 10)
	stack := make([]*Dir, 0, 10)
	stack = append(stack, &Dir{Path: dir, Depth: 0, Index: 0})
	indent := ""
	var current *Dir
	var builder strings.Builder
	for len(stack) > 0 {
		current = stack[len(stack)-1]
		if current.Children == nil {
			realPath, err := filepath.EvalSymlinks(current.Path)
			if err != nil {
				realPath = current.Path
			}
			if visited[realPath] {
				stack = stack[:len(stack)-1]
				continue
			}
			visited[realPath] = true
			datas, err := os.ReadDir(current.Path)
			if err != nil {
				stack = stack[:len(stack)-1]
				continue
			}
			current.Children = datas
		}
		if current.Index >= len(current.Children) {
			stack = stack[:len(stack)-1]
			continue
		}
		child := current.Children[current.Index]
		current.Index++
		if instance.ShouldIgnore(child.Name(), ignore) {
			continue
		}
		indent = strings.Repeat(" ", current.Depth+1)
		if child.IsDir() {
			builder.WriteString(fmt.Sprintf("%s- %s/\n", indent, child.Name()))
			stack = append(stack, &Dir{
				Path:  filepath.Join(current.Path, child.Name()),
				Depth: current.Depth + 1,
				Index: 0,
			})
		} else {
			builder.WriteString(fmt.Sprintf("%s- %s\n", indent, child.Name()))
		}
	}
	return builder.String()
}

func (*Tool) ShouldIgnore(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}
