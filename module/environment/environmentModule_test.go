package environment

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewEnvironmentModule(t *testing.T) {
	logger := zap.NewNop()
	bus := createTestEventBus(t)
	defer bus.Close()

	module := NewEnvironmentModule(bus, logger)

	assert.NotNil(t, module)
	assert.Equal(t, bus, module.bus)
	assert.Equal(t, logger, module.logger)
}

func TestEnvironmentModule_ReadCWD(t *testing.T) {
	logger := zap.NewNop()
	bus := createTestEventBus(t)
	defer bus.Close()

	module := NewEnvironmentModule(bus, logger)

	cwd := module.readCWD()

	assert.NotEmpty(t, cwd)
	assert.NotEqual(t, Backup, cwd)

	expectedCwd, _ := os.Getwd()
	assert.Equal(t, expectedCwd, cwd)
}

func TestEnvironmentModule_CheckVersion(t *testing.T) {
	logger := zap.NewNop()
	bus := createTestEventBus(t)
	defer bus.Close()

	module := NewEnvironmentModule(bus, logger)

	version := module.checkVersion()

	assert.NotEmpty(t, version)
	if runtime.GOOS == "linux" {
		assert.NotEqual(t, Backup, version)
	}
}

func TestEnvironmentModule_CheckGit_WithGitRepo(t *testing.T) {
	logger := zap.NewNop()
	bus := createTestEventBus(t)
	defer bus.Close()

	module := NewEnvironmentModule(bus, logger)
	cwd, _ := os.Getwd()

	isGitRepo := module.checkGit(cwd)

	assert.True(t, isGitRepo, "현재 디렉토리는 git repository여야 합니다")
}

func TestEnvironmentModule_CheckGit_WithoutGitRepo(t *testing.T) {
	logger := zap.NewNop()
	bus := createTestEventBus(t)
	defer bus.Close()

	module := NewEnvironmentModule(bus, logger)

	isGitRepo := module.checkGit("/tmp")

	assert.False(t, isGitRepo, "/tmp는 일반적으로 git repository가 아닙니다")
}

func TestEnvironmentModule_UpdateEnvironmentInfo(t *testing.T) {
	logger := zap.NewNop()
	bus := createTestEventBus(t)
	defer bus.Close()

	var capturedEvent *events.Event[dto.EnvironmentUpdateData]
	var wg sync.WaitGroup
	wg.Add(1)

	events.Subscribe(bus, bus.UpdateEnvironmentEvent, constants.Source(999), func(event events.Event[dto.EnvironmentUpdateData]) {
		capturedEvent = &event
		wg.Done()
	})

	module := NewEnvironmentModule(bus, logger)
	module.UpdateEnvironmentInfo()

	wg.Wait()

	require.NotNil(t, capturedEvent, "UpdateEnvironmentEvent가 발생해야 합니다")
	assert.NotEmpty(t, capturedEvent.Data.CreateID)
	assert.NotEmpty(t, capturedEvent.Data.Cwd)
	assert.Equal(t, runtime.GOOS, capturedEvent.Data.OS)
	assert.NotEmpty(t, capturedEvent.Data.OSVersion)
	assert.NotEmpty(t, capturedEvent.Data.TodayDate)
	assert.Equal(t, constants.EnvironmentModule, capturedEvent.Source)
	assert.WithinDuration(t, time.Now(), capturedEvent.TimeStamp, time.Second)
}

func TestEnvironmentModule_Subscribe_HandlesRequestEvent(t *testing.T) {
	logger := zap.NewNop()
	bus := createTestEventBus(t)
	defer bus.Close()

	var capturedUpdateEvent *events.Event[dto.EnvironmentUpdateData]
	var wg sync.WaitGroup
	wg.Add(1)

	events.Subscribe(bus, bus.UpdateEnvironmentEvent, constants.Source(998), func(event events.Event[dto.EnvironmentUpdateData]) {
		capturedUpdateEvent = &event
		wg.Done()
	})

	NewEnvironmentModule(bus, logger)

	requestEvent := events.Event[dto.EnvironmentRequestData]{
		Data: dto.EnvironmentRequestData{
			CreateID: types.NewCreateID(),
		},
		TimeStamp: time.Now(),
		Source:    constants.Source(997),
	}

	events.Publish(bus, bus.RequestEnvironmentEvent, requestEvent)

	wg.Wait()

	require.NotNil(t, capturedUpdateEvent, "RequestEnvironmentEvent 처리 후 UpdateEnvironmentEvent가 발생해야 합니다")
	assert.NotEmpty(t, capturedUpdateEvent.Data.CreateID)
	assert.Equal(t, constants.EnvironmentModule, capturedUpdateEvent.Source)
}

func TestEnvironmentModule_Subscribe_Integration(t *testing.T) {
	logger := zap.NewNop()
	bus := createTestEventBus(t)
	defer bus.Close()

	var receivedEvents []events.Event[dto.EnvironmentUpdateData]
	var mu sync.Mutex
	var wg sync.WaitGroup

	events.Subscribe(bus, bus.UpdateEnvironmentEvent, constants.Source(996), func(event events.Event[dto.EnvironmentUpdateData]) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		wg.Done()
	})

	NewEnvironmentModule(bus, logger)

	numEvents := 3
	wg.Add(numEvents)

	for i := 0; i < numEvents; i++ {
		requestEvent := events.Event[dto.EnvironmentRequestData]{
			Data: dto.EnvironmentRequestData{
				CreateID: types.NewCreateID(),
			},
			TimeStamp: time.Now(),
			Source:    constants.Source(995),
		}
		events.Publish(bus, bus.RequestEnvironmentEvent, requestEvent)
	}

	wg.Wait()

	mu.Lock()
	assert.Len(t, receivedEvents, numEvents, "모든 요청 이벤트에 대해 업데이트 이벤트가 발생해야 합니다")
	mu.Unlock()

	for _, event := range receivedEvents {
		assert.NotEmpty(t, event.Data.CreateID)
		assert.NotEmpty(t, event.Data.Cwd)
		assert.Equal(t, runtime.GOOS, event.Data.OS)
		assert.Equal(t, constants.EnvironmentModule, event.Source)
	}
}

func createTestEventBus(t *testing.T) *events.EventBus {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := events.NewEventBus(config, logger)
	require.NoError(t, err)
	require.NotNil(t, bus)

	return bus
}
