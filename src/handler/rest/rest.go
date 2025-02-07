package rest

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/downsized-devs/sdk-go/appcontext"
	"github.com/downsized-devs/sdk-go/auth"
	"github.com/downsized-devs/sdk-go/configreader"
	"github.com/downsized-devs/sdk-go/logger"
	"github.com/downsized-devs/sdk-go/parser"
	"github.com/downsized-devs/template-service-go/docs/swagger"
	"github.com/downsized-devs/template-service-go/src/business/usecase"
	"github.com/downsized-devs/template-service-go/src/handler/scheduler"
	"github.com/downsized-devs/template-service-go/src/utils/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gopkg.in/yaml.v2"
)

const (
	infoRequest  string = `httpclient Sent Request: uri=%v method=%v`
	infoResponse string = `httpclient Received Response: uri=%v method=%v resp_code=%v`
)

var once = &sync.Once{}

type REST interface {
	Run()
}

type rest struct {
	http         *gin.Engine
	conf         config.GinConfig
	configreader configreader.Interface
	auth         auth.Interface
	json         parser.JsonInterface
	log          logger.Interface
	uc           *usecase.Usecases
	scheduler    scheduler.Interface
}

type InitParam struct {
	Conf         config.GinConfig
	Configreader configreader.Interface
	Log          logger.Interface
	Auth         auth.Interface
	Json         parser.JsonInterface
	Uc           *usecase.Usecases
	Scheduler    scheduler.Interface
}

func Init(params InitParam) REST {
	r := &rest{}
	once.Do(func() {

		switch params.Conf.Mode {
		case gin.ReleaseMode:
			gin.SetMode(gin.ReleaseMode)
		case gin.DebugMode, gin.TestMode:
			gin.SetMode(gin.TestMode)
		default:
			gin.SetMode("")
		}

		httpServer := gin.New()

		r = &rest{
			conf:         params.Conf,
			configreader: params.Configreader,
			log:          params.Log,
			auth:         params.Auth,
			json:         params.Json,
			http:         httpServer,
			uc:           params.Uc,
			scheduler:    params.Scheduler,
		}

		// Set CORS
		switch r.conf.CORS.Mode {
		case "allowall":
			r.http.Use(cors.New(cors.Config{
				AllowAllOrigins: true,
				AllowHeaders:    []string{"*"},
				AllowMethods: []string{
					http.MethodHead,
					http.MethodGet,
					http.MethodPost,
					http.MethodPut,
					http.MethodPatch,
					http.MethodDelete,
				},
			}))
		default:
			r.http.Use(cors.New(cors.DefaultConfig()))
		}

		// Set Recovery
		r.http.Use(gin.Recovery())

		// Set Timeout
		r.http.Use(r.SetTimeout)

		r.Register()
	})

	return r
}

func (r *rest) Run() {
	// Create context that listens for the interrupt signal from the OS.
	c := appcontext.SetServiceVersion(context.Background(), r.conf.Meta.Version)
	ctx, stop := signal.NotifyContext(c, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := ":8080"
	if r.conf.Port != "" {
		port = fmt.Sprintf(":%s", r.conf.Port)
	}

	srv := &http.Server{
		Addr:              port,
		Handler:           r.http,
		ReadHeaderTimeout: 2 * time.Second,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.log.Error(ctx, fmt.Sprintf("Serving HTTP error: %s", err.Error()))
		}
	}()
	r.log.Info(ctx, fmt.Sprintf("Listening and Serving HTTP on %s", srv.Addr))

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	r.log.Info(ctx, "Shutting down server...")

	// The context is used to inform the server it has timeout duration to finish
	// the request it is currently handling
	quitctx, cancel := context.WithTimeout(c, r.conf.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(quitctx); err != nil {
		r.log.Fatal(quitctx, fmt.Sprintf("Server Shutdown: %s", err.Error()))
	}
	r.log.Info(quitctx, "Server Shut Down.")
}

func (r *rest) Register() {
	// server health and testing purpose
	r.http.GET("/ping", r.Ping)
	r.registerSwaggerRoutes()
	r.registerPlatformRoutes()

	commonPrivateMiddlewares := gin.HandlersChain{
		r.addFieldsToContext, r.BodyLogger,
	}

	// register middlewares
	v1 := r.http.Group("/v1/", commonPrivateMiddlewares...)

	// scheduler
	v1.POST("/admin/scheduler/trigger", r.TriggerScheduler)
}

func (r *rest) registerSwaggerRoutes() {
	if r.conf.Swagger.Enabled {
		swagger.SwaggerInfo.Title = r.conf.Meta.Title
		swagger.SwaggerInfo.Description = r.conf.Meta.Description
		swagger.SwaggerInfo.Version = r.conf.Meta.Version
		swagger.SwaggerInfo.Host = r.conf.Meta.Host
		swagger.SwaggerInfo.BasePath = r.conf.Meta.BasePath

		swaggerAuth := gin.Accounts{
			r.conf.Swagger.BasicAuth.Username: r.conf.Swagger.BasicAuth.Password,
		}

		r.http.GET(fmt.Sprintf("%s/*any", r.conf.Swagger.Path),
			gin.BasicAuthForRealm(swaggerAuth, "Restricted"),
			ginSwagger.WrapHandler(swaggerfiles.Handler))
	}
}

func (r *rest) registerPlatformRoutes() {
	if r.conf.Platform.Enabled {
		platformAuth := gin.Accounts{
			r.conf.Platform.BasicAuth.Username: r.conf.Platform.BasicAuth.Password,
		}

		r.http.GET(r.conf.Platform.Path,
			gin.BasicAuthForRealm(platformAuth, "Restricted"),
			r.platformConfig)
	}
}

func (r *rest) platformConfig(ctx *gin.Context) {
	conf := r.configreader.AllSettings()

	switch ctx.Query("output") {
	case "yaml":
		c, err := yaml.Marshal(conf)
		if err != nil {
			r.httpRespError(ctx, err)
			return
		}
		ctx.String(http.StatusOK, string(c))
	default:
		ctx.IndentedJSON(http.StatusOK, conf)
	}
}
