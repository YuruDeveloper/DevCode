package dto

import "github.com/google/uuid"

type StreamStartData struct {
	RequestUUID uuid.UUID
}

type StreamChunkData struct {
	RequestUUID uuid.UUID
	Content     string
	IsComplete  bool
}

type StreamCompleteData struct {
	RequestUUID  uuid.UUID
	FinalMessage string
	IsComplete   bool
}

type StreamErrorData struct {
	RequestUUID uuid.UUID
	Error       error
	ChunkCount  int
}

type StreamCancelData struct {
	RequestUUID uuid.UUID
}
