package events

import (
	"UniCode/src/types"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

const TestServiceType types.Source = 999

// TestSubscriber is a test implementation of EventHandler interface
type TestSubscriber struct {
	ID             types.Source
	ReceivedEvents []Event
	Mutex          sync.Mutex
}

func (subscriber *TestSubscriber) HandleEvent(event Event) {
	subscriber.Mutex.Lock()
	defer subscriber.Mutex.Unlock()
	subscriber.ReceivedEvents = append(subscriber.ReceivedEvents, event)
}

func (subscriber *TestSubscriber) GetID() types.Source {
	return subscriber.ID
}

func (subscriber *TestSubscriber) GetReceivedEvents() []Event {
	subscriber.Mutex.Lock()
	defer subscriber.Mutex.Unlock()
	return append([]Event{}, subscriber.ReceivedEvents...)
}

// SetupEventBus creates a new EventBus for testing
func SetupEventBus() *EventBus {
	return NewEventBus()
}

// CreateTestSubscriber creates a test subscriber with given ID
func CreateTestSubscriber(ID types.Source) *TestSubscriber {
	return &TestSubscriber{ID: ID}
}

// CreateTestEvent creates a test event for testing
func CreateTestEvent(EventType EventType, Source types.Source, Message string) Event {
	return Event{
		Type: EventType,
		Data: types.RequestData{
			SessionUUID: uuid.New(),
			RequestUUID: uuid.New(),
			Message:     Message,
		},
		Timestamp: time.Now(),
		Source:    Source,
	}
}

func TestNewEventBus_ShouldCreateEventBusSuccessfully(t *testing.T) {
	// When
	EventBus := NewEventBus()

	// Then
	if EventBus == nil {
		t.Fatal("NewEventBus는 nil을 반환하면 안됩니다")
	}

	if EventBus.Subscribers == nil {
		t.Error("Subscribers 맵이 초기화되지 않았습니다")
	}
}

func TestEventBus_Subscribe_ShouldAddSubscriberSuccessfully(t *testing.T) {
	// Given
	EventBus := SetupEventBus()
	TestSubscriber := CreateTestSubscriber(TestServiceType)

	// When
	EventBus.Subscribe(UserInputEvent, TestSubscriber)

	// Then
	Subscribers := EventBus.Subscribers[UserInputEvent]
	if len(Subscribers) != 1 {
		t.Errorf("구독자 수 = %d, 예상값 1", len(Subscribers))
	}

	if Subscribers[0].GetID() != TestServiceType {
		t.Error("구독자 ID가 올바르지 않습니다")
	}
}

func TestEventBus_UnSubscribe_ShouldRemoveSubscriberSuccessfully(t *testing.T) {
	// Given
	EventBus := SetupEventBus()
	TestSubscriber := CreateTestSubscriber(TestServiceType)
	EventBus.Subscribe(UserInputEvent, TestSubscriber)

	// When
	EventBus.UnSubscribe(UserInputEvent, TestServiceType)

	// Then
	Subscribers := EventBus.Subscribers[UserInputEvent]
	if len(Subscribers) != 0 {
		t.Errorf("구독자 수 = %d, 예상값 0", len(Subscribers))
	}
}

func TestEventBus_Publish_ShouldDeliverEventToSubscriber(t *testing.T) {
	// Given
	EventBus := SetupEventBus()
	TestSubscriber := CreateTestSubscriber(TestServiceType)
	EventBus.Subscribe(UserInputEvent, TestSubscriber)
	TestEvent := CreateTestEvent(UserInputEvent, TestServiceType, "테스트 메시지")

	// When
	EventBus.Publish(TestEvent)
	time.Sleep(10 * time.Millisecond) // Wait for async processing

	// Then
	ReceivedEvents := TestSubscriber.GetReceivedEvents()
	if len(ReceivedEvents) != 1 {
		t.Errorf("받은 이벤트 수 = %d, 예상값 1", len(ReceivedEvents))
		return
	}

	AssertEventEquals(t, ReceivedEvents[0], TestEvent)
}

func TestEventBus_MultipleSubscribers_ShouldDeliverEventToAllSubscribers(t *testing.T) {
	// Given
	EventBus := SetupEventBus()
	Subscriber1 := CreateTestSubscriber(TestServiceType)
	Subscriber2 := CreateTestSubscriber(types.MessageService)
	
	EventBus.Subscribe(UserInputEvent, Subscriber1)
	EventBus.Subscribe(UserInputEvent, Subscriber2)
	
	TestEvent := CreateTestEvent(UserInputEvent, TestServiceType, "멀티 구독자 테스트")

	// When
	EventBus.Publish(TestEvent)
	time.Sleep(20 * time.Millisecond) // Wait for async processing

	// Then
	ReceivedEvents1 := Subscriber1.GetReceivedEvents()
	ReceivedEvents2 := Subscriber2.GetReceivedEvents()

	if len(ReceivedEvents1) != 1 {
		t.Errorf("구독자1이 받은 이벤트 수 = %d, 예상값 1", len(ReceivedEvents1))
	}

	if len(ReceivedEvents2) != 1 {
		t.Errorf("구독자2가 받은 이벤트 수 = %d, 예상값 1", len(ReceivedEvents2))
	}
}

func TestEventBus_ConcurrentPublish_ShouldHandleAllEventsCorrectly(t *testing.T) {
	// Given
	EventBus := SetupEventBus()
	TestSubscriber := CreateTestSubscriber(TestServiceType)
	EventBus.Subscribe(UserInputEvent, TestSubscriber)

	NumberOfEvents := 10
	var WaitGroup sync.WaitGroup

	// When
	for i := 0; i < NumberOfEvents; i++ {
		WaitGroup.Add(1)
		go func(index int) {
			defer WaitGroup.Done()
			TestEvent := CreateTestEvent(UserInputEvent, TestServiceType, "동시성 테스트")
			EventBus.Publish(TestEvent)
		}(i)
	}

	WaitGroup.Wait()
	time.Sleep(50 * time.Millisecond) // Wait for all events to be processed

	// Then
	ReceivedEvents := TestSubscriber.GetReceivedEvents()
	if len(ReceivedEvents) != NumberOfEvents {
		t.Errorf("받은 이벤트 수 = %d, 예상값 %d", len(ReceivedEvents), NumberOfEvents)
	}
}

// AssertEventEquals compares two events for equality
func AssertEventEquals(t *testing.T, actual, expected Event) {
	if actual.Type != expected.Type {
		t.Error("받은 이벤트 타입이 올바르지 않습니다")
	}

	if actual.Source != expected.Source {
		t.Error("받은 이벤트 소스가 올바르지 않습니다")
	}
}