package dto

import "github.com/google/uuid"

type EnvironmentUpdateData struct {
	CreateUUID         uuid.UUID
	Cwd                string
	OS                 string
	OSVersion          string
	IsDirectoryGitRepo bool
	TodayDate          string
}

type EnvironmentRequestData struct {
	CreateUUID uuid.UUID
}
