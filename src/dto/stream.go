package dto

import (
	"DevCode/src/types"
)

type StreamStartData struct {
	RequestID types.RequestID
}

type StreamChunkData struct {
	RequestID  types.RequestID
	Content    string
	IsComplete bool
}

type StreamCompleteData struct {
	RequestID    types.RequestID
	FinalMessage string
	IsComplete   bool
}

type StreamErrorData struct {
	RequestID  types.RequestID
	Error      error
	ChunkCount int
}

type StreamCancelData struct {
	RequestID types.RequestID
}
