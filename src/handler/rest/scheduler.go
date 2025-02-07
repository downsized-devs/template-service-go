package rest

import (
	"github.com/downsized-devs/sdk-go/codes"
	"github.com/downsized-devs/template-service-go/src/business/entity"
	"github.com/gin-gonic/gin"
)

// @Summary Trigger Scheduler
// @Description Trigger Scheduler
// @Security BearerAuth
// @Tags Scheduler
// @Param trigger_input body entity.TriggerSchedulerParams true "Parameter for triggering scheduler"
// @Produce json
// @Success 200 {object} entity.HTTPResp{}
// @Failure 500 {object} entity.HTTPResp{}
// @Failure 401 {object} entity.HTTPResp{}
// @Failure 404 {object} entity.HTTPResp{}
// @Router /v1/admin/scheduler/trigger [POST]
func (r *rest) TriggerScheduler(ctx *gin.Context) {
	triggerParams := entity.TriggerSchedulerParams{}
	if err := r.Bind(ctx, &triggerParams); err != nil {
		r.httpRespError(ctx, err)
		return
	}

	if err := r.scheduler.TriggerScheduler(triggerParams.Name); err != nil {
		r.httpRespError(ctx, err)
		return
	}

	r.httpRespSuccess(ctx, codes.CodeSuccess, nil, nil)
}
