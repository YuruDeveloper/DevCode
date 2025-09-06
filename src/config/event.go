package config

const (
	BackupPoolSize = 10000
)

type EventBusConfig struct {
	PoolSize int
}

func (instance *EventBusConfig) Default() {
	if instance.PoolSize == 0 {
		instance.PoolSize = BackupPoolSize
	}
}
