package config

const (
	BackupDot        = "●"
	BackupSelectChar = ">"
)

type ViewConfig struct {
	Dot        string
	SelectChar string
}

func (instance *ViewConfig) Default() {
	if instance.Dot == "" {
		instance.Dot = BackupDot
	}
	if instance.SelectChar == "" {
		instance.SelectChar = BackupSelectChar
	}
}
