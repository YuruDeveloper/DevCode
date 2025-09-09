package dto

import (
	"DevCode/constants"
	"DevCode/types"
)

type UserRequestData struct {
	SessionID types.SessionID
	RequestID types.RequestID
	Message   string
}

type UserDecisionData struct {
	RequestID  types.RequestID
	ToolCallID types.ToolCallID
	Accept     bool
}

type UpdateUserStatusData struct {
	Status constants.UserStatus
}

type UpdateViewData struct{}
