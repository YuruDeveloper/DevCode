package dto

import "github.com/google/uuid"

type ParsedChunkData struct {
	RequestUUID uuid.UUID
	Content     string
	IsComplete  bool
}

type ParsedChunkErrorData struct {
	RequestUUID uuid.UUID
	Error       string
}
