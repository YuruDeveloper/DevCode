package devcodeerror

type ErrorCode uint

const (
	FailLoggerSetup = ErrorCode(100 + iota)
	FailReadConfig
	FailCreateEventBus
)

const (
	FailRunApp = ErrorCode(200 + iota)
	FailHandleEvent
)

const (
	FailReadEnvironment = ErrorCode(300 + iota)
)
