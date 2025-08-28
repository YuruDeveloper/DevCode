package environment

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/service"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/google/uuid"
)

func NewEnvironmentService(bus events.Bus) *EnvironmentService {
	service := &EnvironmentService{bus: bus}
	bus.Subscribe(events.RequestEnvironmentEvent, service)
	return service
}

type EnvironmentService struct {
	bus    events.Bus
}

func (instance *EnvironmentService) HandleEvent(event events.Event) {
	if event.Type == events.RequestEnvironmentEvent {
		cwd, _ := os.Getwd()
		cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-dir")
		gitErr := cmd.Run()
		cmd = exec.Command("uname", "-r")
		cmd.Run()
		version, _ := cmd.Output()
		service.PublishEvent(instance.bus, events.UpdateEnvironmentEvent,
			dto.EnvironmentUpdateData{
				CreateUUID:         uuid.New(),
				Cwd:                cwd,
				OS:                 runtime.GOOS,
				OSVersion:          string(version),
				IsDirectoryGitRepo: gitErr == nil,
				TodayDate:          time.Now().Format("2006-01-02"),
			},
			constants.EnvironmentService)
	}
}

func (instance *EnvironmentService) GetID() constants.Source {
	return constants.EnvironmentService
}
