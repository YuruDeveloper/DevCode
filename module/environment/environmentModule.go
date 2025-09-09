package environment

import (
	devcodeerror "DevCode/DevCodeError"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	Backup = "Unknown"
)

func NewEnvironmentModule(bus *events.EventBus, logger *zap.Logger) *EnvironmentModule {
	module := &EnvironmentModule{bus: bus, logger: logger}
	module.Subscribe()
	return module
}

type EnvironmentModule struct {
	bus    *events.EventBus
	logger *zap.Logger
}

func (instance *EnvironmentModule) Subscribe() {
	events.Subscribe(instance.bus, instance.bus.RequestEnvironmentEvent,
		constants.EnvironmentModule,
		func(event events.Event[dto.EnvironmentRequestData]) {
			instance.UpdateEnvironmentInfo()
		},
	)
}

func (instance *EnvironmentModule) readCWD() string {
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

func (instance *EnvironmentModule) checkGit(cwd string) bool {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-dir")
	if gitErr := cmd.Run(); gitErr != nil {
		instance.logger.Warn("", zap.Error(devcodeerror.Wrap(gitErr, devcodeerror.FailReadEnvironment, "Fail run git cmd")))
		return false
	}
	return true
}

func (instance *EnvironmentModule) checkVersion() string {
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

func (instance *EnvironmentModule) UpdateEnvironmentInfo() {
	cwd := instance.readCWD()
	git := instance.checkGit(cwd)
	version := instance.checkVersion()
	events.Publish(instance.bus, instance.bus.UpdateEnvironmentEvent, events.Event[dto.EnvironmentUpdateData]{
		Data: dto.EnvironmentUpdateData{
			CreateID:           types.NewCreateID(),
			Cwd:                cwd,
			OS:                 runtime.GOOS,
			OSVersion:          version,
			IsDirectoryGitRepo: git,
			TodayDate:          time.Now().Format("2006-01-02"),
		},
		TimeStamp: time.Now(),
		Source:    constants.EnvironmentModule,
	})
}
