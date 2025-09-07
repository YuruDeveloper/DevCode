package environment

import (
	devcodeerror "DevCode/src/DevCodeError"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/types"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"go.uber.org/zap"
)

const (
	Backup = "Unknown"
)

func NewEnvironmentService(bus *events.EventBus, logger *zap.Logger) *EnvironmentService {
	service := &EnvironmentService{bus: bus, logger: logger}
	service.Subscribe()
	return service
}

type EnvironmentService struct {
	bus    *events.EventBus
	logger *zap.Logger
}

func (instance *EnvironmentService) Subscribe() {
	instance.bus.RequestEnvironmentEvent.Subscribe(
		constants.EnvironmentService,
		func(event events.Event[dto.EnvironmentRequestData]) {
			instance.UpdateEnvironmentInfo()
		},
	)
}

func (instance *EnvironmentService) readCWD() string {
	cwd, err := os.Getwd()
	if err != nil {
		instance.logger.Warn("", zap.Error(devcodeerror.Wrap(
			err,
			devcodeerror.FailReadEnvironment,
			"Fail to Read Cwd",
		)))
		cwd = Backup
	}

	return cwd
}

func (instance *EnvironmentService) checkGit(cwd string) bool {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-dir")
	if gitErr := cmd.Run(); gitErr != nil {
		instance.logger.Warn("", zap.Error(devcodeerror.Wrap(gitErr, devcodeerror.FailReadEnvironment, "Fail run git cmd")))
		return false
	}
	return true
}

func (instance *EnvironmentService) checkVersion() string {
	var version string
	cmd := exec.Command("uname", "-r")
	result, err := cmd.Output()
	if err != nil {
		instance.logger.Warn("", zap.Error(devcodeerror.Wrap(err, devcodeerror.FailReadEnvironment, "Fail run uname cmd")))
		version = Backup
	} else {
		version = strings.TrimSpace(string(result))
	}
	return version
}

func (instance *EnvironmentService) UpdateEnvironmentInfo() {
	cwd := instance.readCWD()
	git := instance.checkGit(cwd)
	version := instance.checkVersion()
	instance.bus.UpdateEnvironmentEvent.Publish(events.Event[dto.EnvironmentUpdateData]{
		Data: dto.EnvironmentUpdateData{
			CreateID:       	types.NewCreateID(),
			Cwd:                cwd,
			OS:                 runtime.GOOS,
			OSVersion:          version,
			IsDirectoryGitRepo: git,
			TodayDate:          time.Now().Format("2006-01-02"),
		},
		TimeStamp: time.Now(),
		Source:    constants.EnvironmentService,
	})
}
