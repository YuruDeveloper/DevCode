package config

const (
	BackupName    = "DevCode"
	BackupVersion = "0.0.1"
)

type McpServiceConfig struct {
	Name          string
	Version       string
	ServerName    string
	ServerVersion string
}

func (instance *McpServiceConfig) Default() {
	if instance.Name == "" {
		instance.Name = BackupName
	}
	if instance.Version == "" {
		instance.Version = BackupVersion
	}
	if instance.ServerName == "" {
		instance.ServerName = BackupName
	}
	if instance.ServerVersion == "" {
		instance.ServerVersion = BackupVersion
	}
}
