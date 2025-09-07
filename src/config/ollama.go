package config

import (
	"net/url"
	"os"
)

const (
	BackupMessageLimit               = 100
	BackupDefaultSystemMessageLength = 10
	BackupEnviromentInfo             = "Here is useful information about the environment you are running in:\n"
	BackupDefaultToolSize            = 10
	BackupDefaultRequestContentsSize = 10
	BackupToolCallSize               = 5
	BackupDefaultActiveStreamSzie    = 10
	BackupUrl                        = "http://127.0.0.1:11434"
	BackupModel                      = "llama3.1:8b"
	BackupPrompt                     = "You are DevCode : Code Asstance"
)

type OllamaServiceConfig struct {
	MessageLimit               int
	DefaultSystemMessageLength int
	EnvironmentInfo            string
	DefaultToolSize            int
	DefaultRequestContentsSize int
	DefaultToolCallSize        int
	urlText                    string
	Url                        *url.URL
	Model                      string
	system                     string
	Prompt                     string
	DefaultActiveStreamSize    int
}

func (instance *OllamaServiceConfig) Default() {
	if instance.MessageLimit == 0 {
		instance.MessageLimit = BackupMessageLimit
	}
	if instance.DefaultSystemMessageLength == 0 {
		instance.DefaultActiveStreamSize = BackupDefaultSystemMessageLength
	}
	if instance.EnvironmentInfo == "" {
		instance.EnvironmentInfo = BackupEnviromentInfo
	}
	if instance.DefaultToolSize == 0 {
		instance.DefaultToolSize = BackupDefaultToolSize
	}
	if instance.DefaultRequestContentsSize == 0 {
		instance.DefaultRequestContentsSize = BackupDefaultRequestContentsSize
	}
	if instance.DefaultToolCallSize == 0 {
		instance.DefaultToolCallSize = BackupToolCallSize
	}
	if instance.DefaultActiveStreamSize == 0 {
		instance.DefaultActiveStreamSize = BackupDefaultActiveStreamSzie
	}
	if instance.urlText == "" {
		instance.urlText = BackupUrl
	}
	var parsed *url.URL
	var err error
	if parsed, err = url.Parse(instance.urlText); err != nil {
		parsed, _ = url.Parse(BackupUrl)
	}
	instance.Url = parsed
	if instance.Model == "" {
		instance.Model = BackupModel
	}
	if instance.system == "" {
		instance.Prompt = ""
	}
	systemPrompt, err := os.ReadFile(instance.system)
	if err != nil {
		instance.Prompt = BackupPrompt
	} else {
		instance.Prompt = string(systemPrompt)
	}
}
