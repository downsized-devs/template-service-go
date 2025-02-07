package config

import (
	"time"

	"github.com/downsized-devs/sdk-go/logger"
	"github.com/downsized-devs/sdk-go/parser"
	"github.com/downsized-devs/sdk-go/sql"
)

type Application struct {
	Log       logger.Config
	Gin       GinConfig
	SQL       sql.Config
	Parser    parser.Options
	Scheduler SchedulerConfig
}

type GinConfig struct {
	Port            string
	Mode            string
	LogRequest      bool
	LogResponse     bool
	Timeout         time.Duration
	ShutdownTimeout time.Duration
	CORS            CORSConfig
	Meta            GinMeta
	Swagger         SwaggerConfig
	Platform        PlatformConfig
}

type GinMeta struct {
	Title       string
	Description string
	Host        string
	BasePath    string
	Version     string
}

type CORSConfig struct {
	Mode string
}
type SwaggerConfig struct {
	Enabled   bool
	Path      string
	BasicAuth BasicAuthConf
}

type PlatformConfig struct {
	Enabled   bool
	Path      string
	BasicAuth BasicAuthConf
}

type BasicAuthConf struct {
	Username string
	Password string
}

type SchedulerTaskConf struct {
	Name          string
	Enabled       bool
	TimeType      string
	Interval      time.Duration
	ScheduledTime string
}

type SchedulerConfig struct {
	HelloWorld SchedulerTaskConf
}

func Init() Application {
	return Application{}
}
