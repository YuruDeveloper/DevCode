package utils

import (
	"DevCode/dto"
	"fmt"
	"strings"
)

func EnvironmentUpdateDataToString(data dto.EnvironmentUpdateData) string {
	var builder strings.Builder
	builder.WriteString("<env>\n")
	builder.WriteString(fmt.Sprintf("Working directory: %s\n", data.Cwd))
	builder.WriteString(fmt.Sprintf("Is directory a git repo: %t\n", data.IsDirectoryGitRepo))
	builder.WriteString(fmt.Sprintf("Platform: %s\n", data.OS))
	builder.WriteString(fmt.Sprintf("OS Version: %s\n", data.OSVersion))
	builder.WriteString(fmt.Sprintf("Today's date: %s\n", data.TodayDate))
	builder.WriteString("</env>\n")
	return builder.String()
}
