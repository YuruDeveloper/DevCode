package events

import (
	"UniCode/src/types"
	"time"
)

type EventType int 

const (
	UserInputEvent = iota + 1
	ToolCallEvent
	ErrorEvent
	AcceptToolEvent
	AccpetedToolCallEvent
	RejectToolEvnet
	AssistantOutputEvent
	AssistantFinalOutputEvent
	RequestEnvionmentvent
	UpdateEnvionmentEvent
	RequestToolListEvent
	UpdateToolListEvent
	LLMResponseEvent
)


type Event struct {
	Type EventType
	Data any
	Timestamp time.Time
	Source types.Source
}

type Subsciber interface {
	HandleEvent(event Event) 
	GetID() types.Source
}

