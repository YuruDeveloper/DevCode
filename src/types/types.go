package types

import (
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Tool[T any] interface {
	Name() string
	Description() string
	Handler() mcp.ToolHandlerFor[T, any]
}


type RequestID uuid.UUID


func NewRequestID() RequestID {
	return RequestID(uuid.New())
}

func (instance RequestID) String() string {
	return uuid.UUID(instance).String()
}

type ToolCallID uuid.UUID 

func NewTooCallID() ToolCallID {
	return ToolCallID(uuid.New())
}

func (instance ToolCallID) String() string {
	return uuid.UUID(instance).String()
}

type SessionID uuid.UUID

func NewSessionID() SessionID {
	return SessionID(uuid.New())
}

func (instance SessionID) String() string{
	return uuid.UUID(instance).String()
}

type CreateID uuid.UUID

func NewCreateID() CreateID {
	return CreateID(uuid.New())
}

func (instance CreateID) String() string {
	return uuid.UUID(instance).String()
}


