package environment_test

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/service/environment"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type MockEnvironmentUpdateHandler struct {
	ReceivedEvents []events.Event[dto.EnvironmentUpdateData]
}

func NewMockEnvironmentUpdateHandler() *MockEnvironmentUpdateHandler {
	return &MockEnvironmentUpdateHandler{
		ReceivedEvents: make([]events.Event[dto.EnvironmentUpdateData], 0),
	}
}

func (m *MockEnvironmentUpdateHandler) HandleEvent(event events.Event[dto.EnvironmentUpdateData]) {
	m.ReceivedEvents = append(m.ReceivedEvents, event)
}

func TestNewEnvironmentService(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	logger := zap.NewNop()
	bus, err := events.NewEventBus(eventBusConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	service := environment.NewEnvironmentService(bus, logger)

	assert.NotNil(t, service)
}

func TestEnvironmentService_HandleEvent_RequestEnvironmentEvent(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	logger := zap.NewNop()
	bus, err := events.NewEventBus(eventBusConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	// Environment update 이벤트를 받을 핸들러 설정
	updateHandler := NewMockEnvironmentUpdateHandler()
	bus.UpdateEnvironmentEvent.Subscribe(constants.Model, updateHandler.HandleEvent)

	_ = environment.NewEnvironmentService(bus, logger)

	// Request Environment 이벤트 발행
	requestEvent := events.Event[dto.EnvironmentRequestData]{
		Data:      dto.EnvironmentRequestData{},
		TimeStamp: time.Now(),
		Source:    constants.MessageService,
	}

	// Environment Service의 HandleEvent를 직접 호출하는 대신
	// EventBus를 통해 이벤트 발행
	bus.RequestEnvironmentEvent.Publish(requestEvent)

	// 이벤트 처리 대기
	time.Sleep(100 * time.Millisecond)

	require.Len(t, updateHandler.ReceivedEvents, 1)

	publishedEvent := updateHandler.ReceivedEvents[0]
	assert.Equal(t, constants.EnvironmentService, publishedEvent.Source)

	envData := publishedEvent.Data
	assert.NotEmpty(t, envData.CreateID)
	assert.NotEmpty(t, envData.Cwd)
	assert.NotEmpty(t, envData.OS)
	assert.NotEmpty(t, envData.TodayDate)

	// 날짜 형식 검증
	_, err = time.Parse("2006-01-02", envData.TodayDate)
	assert.NoError(t, err)
}

func TestEnvironmentService_EnvironmentData_UniqueUUIDs(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	logger := zap.NewNop()
	bus, err := events.NewEventBus(eventBusConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	updateHandler := NewMockEnvironmentUpdateHandler()
	bus.UpdateEnvironmentEvent.Subscribe(constants.Model, updateHandler.HandleEvent)

	_ = environment.NewEnvironmentService(bus, logger)

	// 두 번의 환경 정보 요청
	for i := 0; i < 2; i++ {
		requestEvent := events.Event[dto.EnvironmentRequestData]{
			Data:      dto.EnvironmentRequestData{},
			TimeStamp: time.Now(),
			Source:    constants.MessageService,
		}
		bus.RequestEnvironmentEvent.Publish(requestEvent)
	}

	// 이벤트 처리 대기
	time.Sleep(200 * time.Millisecond)

	require.Len(t, updateHandler.ReceivedEvents, 2)

	envData1 := updateHandler.ReceivedEvents[0].Data
	envData2 := updateHandler.ReceivedEvents[1].Data

	// 환경 정보는 동일해야 함
	assert.Equal(t, envData1.Cwd, envData2.Cwd)
	assert.Equal(t, envData1.OS, envData2.OS)
	assert.Equal(t, envData1.TodayDate, envData2.TodayDate)

	// UUID는 다르게 생성되어야 함
	assert.NotEqual(t, envData1.CreateID, envData2.CreateID)
}

func TestEnvironmentService_EnvironmentData_Consistency(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	logger := zap.NewNop()
	bus, err := events.NewEventBus(eventBusConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	updateHandler := NewMockEnvironmentUpdateHandler()
	bus.UpdateEnvironmentEvent.Subscribe(constants.Model, updateHandler.HandleEvent)

	_ = environment.NewEnvironmentService(bus, logger)

	requestEvent := events.Event[dto.EnvironmentRequestData]{
		Data:      dto.EnvironmentRequestData{},
		TimeStamp: time.Now(),
		Source:    constants.MessageService,
	}
	bus.RequestEnvironmentEvent.Publish(requestEvent)

	// 이벤트 처리 대기
	time.Sleep(100 * time.Millisecond)

	require.Len(t, updateHandler.ReceivedEvents, 1)

	envData := updateHandler.ReceivedEvents[0].Data

	// OS 정보는 빈 문자열이 아니어야 함
	assert.NotEmpty(t, envData.OS)
	
	// 현재 작업 디렉토리는 빈 문자열이 아니어야 함
	assert.NotEmpty(t, envData.Cwd)
	
	// 날짜는 YYYY-MM-DD 형식이어야 함
	parsedDate, err := time.Parse("2006-01-02", envData.TodayDate)
	require.NoError(t, err)
	
	// 날짜는 오늘과 비슷해야 함 (테스트 실행 시점 기준)
	today := time.Now()
	diff := parsedDate.Sub(today.Truncate(24 * time.Hour))
	assert.True(t, diff >= -24*time.Hour && diff <= 24*time.Hour, 
		"날짜가 오늘과 너무 차이남: %v", envData.TodayDate)
}