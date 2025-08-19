package service

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/google/uuid"
)

func NewEnvironmentService(bus *events.EventBus) *EnvironmentService {
	service := &EnvironmentService{Bus: bus}
	bus.Subscribe(events.RequestEnvironmentEvent, service)
	return service
}

type EnvironmentService struct {
	Bus *events.EventBus
}

func (instance *EnvironmentService) HandleEvent(event events.Event) {
	if event.Type == events.RequestEnvironmentEvent {
		cwd, _ := os.Getwd()
		cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-dir")
		gitErr := cmd.Run()
		cmd = exec.Command("uname", "-r")
		cmd.Run()
		version, _ := cmd.Output()
		PublishEvent(instance.Bus, events.UpdateEnvironmentEvent,
			types.EnvironmentUpdateData{
				CreateUUID:         uuid.New(),
				Cwd:                cwd,
				OS:                 runtime.GOOS,
				OSVersion:          string(version),
				IsDirectoryGitRepo: gitErr == nil,
				TodayDate:          time.Now().Format("2006-01-02"),
			},
			types.EnvironmentService)
	}
}

func (instance *EnvironmentService) GetID() types.Source {
	return types.EnvironmentService
}
