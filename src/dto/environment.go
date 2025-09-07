package dto

import (
	"DevCode/src/types"
)

type EnvironmentUpdateData struct {
	CreateID           types.CreateID
	Cwd                string
	OS                 string
	OSVersion          string
	IsDirectoryGitRepo bool
	TodayDate          string
}

type EnvironmentRequestData struct {
	CreateID types.CreateID
}
