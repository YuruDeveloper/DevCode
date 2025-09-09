package read

import (
	"DevCode/tools"
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ReadDescription = `Reads a file from the local filesystem. You can access any file directly by
   using this tool.\nAssume this tool is able to read all files on the machine. If the User
  provides a path to a file assume that path is valid. It is okay to read a file that does not
  exist; an error will be returned.\n\nUsage:\n- The file_path parameter must be an absolute
  path, not a relative path\n- By default, it reads up to 2000 lines starting from the
  beginning of the file\n- You can optionally specify a line offset and limit (especially handy
   for long files), but it's recommended to read the whole file by not providing these
  parameters\n- Any lines longer than 2000 characters will be truncated\n- Results are returned
   using cat -n format, with line numbers starting at 1\n- <good-exam1ple>This tool allows UniCode to read
   images (eg PNG, JPG, etc). When reading an image file the contents are presented visually as
	UniCode is a multimodal LLM.\n- This tool can read PDF files (.pdf). PDFs are processed
  page by page, extracting both text and visual content for analysis.\n- This tool can read
  Jupyter notebooks (.ipynb files) and returns all cells with their outputs, combining code,
  text, and visualizations.\n- You have the capability to call multiple tools in a single
  response. It is always better to speculatively read multiple files as a batch that are
  potentially useful. \n- You will regularly be asked to read screenshots. If the user provides
   a path to a screenshot ALWAYS use this tool to view the file at the path. This tool will
  work with all temporary file paths like
  /var/folders/123/abc/T/TemporaryItems/NSIRD_screencaptureui_ZfB1tD/Screenshot.png\n-</good-exam1ple> If you
  read a file that exists but has empty contents you will receive a system reminder warning in
  place of file contents.`
	Name = "Read"
)

type Tool struct {
}

func (*Tool) Name() string {
	return Name
}

func (instance *Tool) Description() string {
	return ReadDescription
}

func (instance *Tool) Handler() mcp.ToolHandlerFor[Input, any] {
	return func(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[Input]) (*mcp.CallToolResultFor[any], error) {
		input := params.Arguments
		if input.FilePath == "" || !filepath.IsAbs(input.FilePath) {
			return nil, fmt.Errorf("invalid path format: %s", input.FilePath)
		}
		if _, err := os.Stat(input.FilePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", input.FilePath)
		}
		file, err := os.Open(input.FilePath)
		if err != nil {
			if os.IsPermission(err) {
				return nil, fmt.Errorf("permission denied: %s", input.FilePath)
			}
			return nil, fmt.Errorf("invalid path format: %s", input.FilePath)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		var content strings.Builder
		totalLines := 0
		readLines := 0
		for scanner.Scan() {
			totalLines++
			if totalLines < input.Offset {
				continue
			}
			if input.Limit > 0 && readLines >= input.Limit {
				break
			}
			fmt.Fprintf(&content, "%6d→\t%s\n", totalLines, scanner.Text())
			readLines++
			if totalLines == 2000 {
				break
			}
		}
		return tools.TextReturn(content.String())
	}
}
