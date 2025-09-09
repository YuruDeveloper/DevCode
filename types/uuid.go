package types

import "github.com/google/uuid"

type RequestID uuid.UUID

func NewRequestID() RequestID {
	return RequestID(uuid.New())
}

func (instance RequestID) String() string {
	return uuid.UUID(instance).String()
}

func (instance RequestID) IsNil() bool {
	return uuid.UUID(instance) == uuid.Nil
}

type ToolCallID uuid.UUID

func NewToolCallID() ToolCallID {
	return ToolCallID(uuid.New())
}

func (instance ToolCallID) String() string {
	return uuid.UUID(instance).String()
}

func (instance ToolCallID) IsNil() bool {
	return uuid.UUID(instance) == uuid.Nil
}

type SessionID uuid.UUID

func NewSessionID() SessionID {
	return SessionID(uuid.New())
}

func (instance SessionID) String() string {
	return uuid.UUID(instance).String()
}

func (instance SessionID) IsNil() bool {
	return uuid.UUID(instance) == uuid.Nil
}

type CreateID uuid.UUID

func NewCreateID() CreateID {
	return CreateID(uuid.New())
}

func (instance CreateID) String() string {
	return uuid.UUID(instance).String()
}

func (instance CreateID) IsNil() bool {
	return uuid.UUID(instance) == uuid.Nil
}
