package environment

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/google/uuid"
)

func NewEnvironmentService(bus *events.EventBus) *EnvironmentService {
	service := &EnvironmentService{bus: bus}
	service.Subscribe()
	return service
}

type EnvironmentService struct {
	bus *events.EventBus
}

func (instance *EnvironmentService) Subscribe() {
	instance.bus.RequestEnvironmentEvent.Subscribe(
		constants.EnvironmentService,
		func(event events.Event[dto.EnvironmentRequestData]) {
			instance.UpdateEnviromentInfo()
		},
	)
}

func (instance *EnvironmentService) UpdateEnviromentInfo() {
	cwd, _ := os.Getwd()
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-dir")
	gitErr := cmd.Run()
	cmd = exec.Command("uname", "-r")
	cmd.Run()
	version, _ := cmd.Output()
	instance.bus.UpdateEnvironmentEvent.Publish(events.Event[dto.EnvironmentUpdateData]{
		Data: dto.EnvironmentUpdateData{
			CreateUUID:         uuid.New(),
			Cwd:                cwd,
			OS:                 runtime.GOOS,
			OSVersion:          string(version),
			IsDirectoryGitRepo: gitErr == nil,
			TodayDate:          time.Now().Format("2006-01-02"),
		},
		TimeStamp: time.Now(),
		Source:    constants.EnvironmentService,
	})
}
