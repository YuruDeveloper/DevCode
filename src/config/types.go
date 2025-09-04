package config

type Config struct {
	ViewConfig          ViewConfig
	McpServiceConfig    McpServiceConfig
	OllamaServiceConfig OllamaServiceConfig
	EventBusConfig      EventBusConfig
}

type ViewConfig struct {
	Dot        string
	SelectChar string
}

type McpServiceConfig struct {
	Name          string
	Version       string
	ServerName    string
	ServerVersion string
}

type OllamaServiceConfig struct {
	MessageLimit               int
	DefaultSystemMessageLength int
	EnvironmentInfo            string
	DefaultToolSize            int
	DefaultRequestContentsSize int
	DefaultToolCallSize        int
	Url                        string
	Model                      string
	System                     string
	DefaultActiveStreamSize    int
}

type EventBusConfig struct {
	PoolSize int
}
