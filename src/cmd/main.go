package main

import (
	"net/http"

	"github.com/downsized-devs/sdk-go/auth"
	"github.com/downsized-devs/sdk-go/configbuilder"
	"github.com/downsized-devs/sdk-go/configreader"
	"github.com/downsized-devs/sdk-go/files"
	"github.com/downsized-devs/sdk-go/logger"
	"github.com/downsized-devs/sdk-go/parser"
	"github.com/downsized-devs/sdk-go/sql"
	"github.com/downsized-devs/template-service-go/src/business/domain"
	"github.com/downsized-devs/template-service-go/src/business/usecase"
	"github.com/downsized-devs/template-service-go/src/handler/rest"
	"github.com/downsized-devs/template-service-go/src/handler/scheduler"
	"github.com/downsized-devs/template-service-go/src/utils/config"
)

// @contact.name   Alvin Radeka
// @contact.email  alvin.radeka@gmail.com

// @securitydefinitions.apikey BearerAuth
// @in header
// @name Authorization

const (
	configfile   string = "./etc/cfg/conf.json"
	templatefile string = "./etc/tpl/conf.template.json"
	appnamespace string = "" // insert your app name space
)

func main() {
	// build config file
	if !files.IsExist(configfile) {
		configbuilder.Init(configbuilder.Options{
			Env:                    "dev", // TODO: change later once configbuilder is set up
			TemplateFile:           templatefile,
			ConfigFile:             configfile,
			Namespace:              appnamespace,
			IgnoreEmptyConfigError: true,
		}).BuildConfig()
	}

	// init config
	cfg := config.Init()
	configreader := configreader.Init(configreader.Options{
		ConfigFile: configfile,
	})
	configreader.ReadConfig(&cfg)

	// init logger
	log := logger.Init(cfg.Log)

	// init parser
	parser := parser.InitParser(log, cfg.Parser)

	// init db conn
	db := sql.Init(cfg.SQL, log, nil)

	// init auth
	auth := auth.Init(auth.Config{}, log, parser.JsonParser(), &http.Client{})

	// init all domain
	dom := domain.Init(domain.InitParam{
		Log:    log,
		Db:     db,
		Parser: parser,
	})

	// init all uc
	uc := usecase.Init(usecase.InitParam{
		Log:    log,
		Parser: parser,
		Dom:    dom,
		Auth:   auth,
	})

	// init http server
	r := rest.Init(rest.InitParam{
		Conf:         cfg.Gin,
		Configreader: configreader,
		Log:          log,
		Json:         parser.JsonParser(),
		Uc:           uc,
		Auth:         auth,
	})

	// init scheduler
	sch := scheduler.Init(cfg.Scheduler, log, auth, uc)

	// run scheduler
	sch.Run()

	// run the http server
	r.Run()
}
