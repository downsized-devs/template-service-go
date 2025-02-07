package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/downsized-devs/sdk-go/auth"
	"github.com/downsized-devs/sdk-go/logger"
	"github.com/downsized-devs/template-service-go/src/business/usecase"
	"github.com/downsized-devs/template-service-go/src/utils/config"
	"github.com/go-co-op/gocron"
)

var (
	once = &sync.Once{}
)

type Interface interface {
	Run()
	TriggerScheduler(name string) error
}

type scheduler struct {
	cron *gocron.Scheduler
	conf config.SchedulerConfig
	log  logger.Interface
	auth auth.Interface
	uc   *usecase.Usecases
}

func Init(conf config.SchedulerConfig, log logger.Interface, auth auth.Interface, uc *usecase.Usecases) Interface {
	s := &scheduler{}
	once.Do(func() {
		cron := gocron.NewScheduler(time.UTC)
		cron.TagsUnique()

		s = &scheduler{
			cron: cron,
			conf: conf,
			log:  log,
			auth: auth,
			uc:   uc,
		}

		s.AssignScheduledTasks()
	})
	return s
}

// AssignScheduledTasks will assign task to a specified schedule
func (s *scheduler) AssignScheduledTasks() {
	s.AssignTask(s.conf.HelloWorld, s.HelloWorld)
}

func (s *scheduler) Run() {
	s.cron.StartAsync()
	s.log.Info(context.Background(), "Scheduler is running")
}

func (s *scheduler) TriggerScheduler(name string) error {
	return s.cron.RunByTag(name)
}

func (s *scheduler) HelloWorld(ctx context.Context) error {
	fmt.Println(ctx, "Hello, 世界!")

	return nil
}
