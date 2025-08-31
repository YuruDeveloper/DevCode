package ls

import "os"

type Input struct {
	Path   string   `json:"path" jsonschema:"description:The absolute path to the directory to list (must be absolute, not relative)"`
	Ignore []string `json:"ignore,omitempty" jsonschema:"description:List of glob patterns to ignore"`
}

type Dir struct {
	Path     string
	Depth    int
	Children []os.DirEntry
	Index    int
}
