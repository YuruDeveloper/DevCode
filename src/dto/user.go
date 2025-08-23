package dto

import "github.com/google/uuid"

type UserRequestData struct {
	SessionUUID uuid.UUID
	RequestUUID uuid.UUID
	Message     string
}

type UserDecisionData struct {
	RequestUUID  uuid.UUID
	ToolCallUUID uuid.UUID
	Accept       bool
}
