package dto

import (
	"DevCode/types"
)

type ParsedChunkData struct {
	RequestID  types.RequestID
	Content    string
	IsComplete bool
}

type ParsedChunkErrorData struct {
	RequestID types.RequestID
	Error     string
}
