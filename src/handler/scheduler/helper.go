package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/downsized-devs/sdk-go/appcontext"
	"github.com/downsized-devs/sdk-go/auth"
	"github.com/downsized-devs/sdk-go/codes"
	"github.com/downsized-devs/sdk-go/errors"
	"github.com/downsized-devs/template-service-go/src/business/entity"
	"github.com/downsized-devs/template-service-go/src/utils/config"
	"github.com/google/uuid"
)

const (
	schedulerUserAgent string = "Cron Scheduler : %s"

	schedulerAssignError string = "Assigning Scheduler %s error: %s"

	schedulerRunning       string = "Running scheduler %s is running"
	schedulerDoneError     string = "Running scheduler %s error: %v"
	schedulerDoneSuccess   string = "Running scheduler %s success"
	schedulerTimeExecution string = "Scheduler %s done in %v"

	schedulerTimeTypeExact    string = "daily"
	schedulerTimeTypeInterval string = "interval"
)

type handlerFunc func(ctx context.Context) error

func createContext(conf config.SchedulerTaskConf) context.Context {
	ctx := context.Background()
	ctx = appcontext.SetUserAgent(ctx, fmt.Sprintf(schedulerUserAgent, conf.Name))
	ctx = appcontext.SetRequestId(ctx, uuid.New().String())
	ctx = appcontext.SetRequestStartTime(ctx, time.Now())
	return ctx
}

func (s *scheduler) AssignTask(conf config.SchedulerTaskConf, task handlerFunc) {
	if conf.Enabled {
		var err error
		ctx := context.Background()
		schedulerFunc := s.taskWrapper(conf, task)

		switch conf.TimeType {
		case schedulerTimeTypeInterval:
			_, err = s.cron.Every(conf.Interval).Tag(conf.Name).Do(schedulerFunc)
		case schedulerTimeTypeExact:
			_, err = s.cron.Every(1).Day().Tag(conf.Name).At(conf.ScheduledTime).Do(schedulerFunc)
		default:
			err = errors.NewWithCode(codes.CodeInternalServerError, "Unknown Scheduler Task Time Type")
		}

		if err != nil {
			s.log.Fatal(ctx, fmt.Sprintf(schedulerAssignError, conf.Name, err.Error()))
		}

	}
}

func (s *scheduler) taskWrapper(conf config.SchedulerTaskConf, task handlerFunc) func() {
	return func() {
		ctx := s.createContext(conf)
		s.log.Info(ctx, fmt.Sprintf(schedulerRunning, conf.Name))
		if err := task(ctx); err != nil {
			s.log.Error(ctx, fmt.Sprintf(schedulerDoneError, conf.Name, err))
		} else {
			s.log.Info(ctx, fmt.Sprintf(schedulerDoneSuccess, conf.Name))
		}

		startTime := appcontext.GetRequestStartTime(ctx)
		s.log.Info(ctx, fmt.Sprintf(schedulerTimeExecution, conf.Name, time.Since(startTime)))
	}
}

func (s *scheduler) createContext(conf config.SchedulerTaskConf) context.Context {
	ctx := context.Background()
	ctx = appcontext.SetUserAgent(ctx, fmt.Sprintf(schedulerUserAgent, conf.Name))
	ctx = appcontext.SetRequestId(ctx, uuid.New().String())
	ctx = appcontext.SetRequestStartTime(ctx, time.Now())

	schedulerUser := auth.UserAuthParam{
		User: auth.User{
			ID:   entity.SystemID,
			Name: entity.SystemName,
		},
	}
	ctx = s.auth.SetUserAuthInfo(ctx, schedulerUser)

	return ctx
}
