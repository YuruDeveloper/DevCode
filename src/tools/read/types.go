package read

type Input struct {
	FilePath string `json:"file_path" jsonschema:"description:The absolute path to the file to read"`
	Offset   int    `json:"offset" jsonschema:"description:The line number to start reading from. Only provide if the file is too large to read at once"`
	Limit    int    `json:"limit" jsonschema:"description:The number of lines to read. Only provide if the file is too large to read at once"`
}
