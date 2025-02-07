package rest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/downsized-devs/sdk-go/appcontext"
	"github.com/downsized-devs/sdk-go/codes"
	"github.com/downsized-devs/sdk-go/errors"
	"github.com/downsized-devs/sdk-go/header"
	"github.com/downsized-devs/template-service-go/src/business/entity"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
)

func (r *rest) BodyLogger(ctx *gin.Context) {
	if r.conf.LogRequest {
		r.log.Info(ctx.Request.Context(),
			fmt.Sprintf(infoRequest, ctx.Request.RequestURI, ctx.Request.Method))
	}

	ctx.Next()
	if r.conf.LogResponse {
		if ctx.Writer.Status() < 300 {
			r.log.Info(ctx.Request.Context(),
				fmt.Sprintf(infoResponse, ctx.Request.RequestURI, ctx.Request.Method, ctx.Writer.Status()))
		} else {
			r.log.Error(ctx.Request.Context(),
				fmt.Sprintf(infoResponse, ctx.Request.RequestURI, ctx.Request.Method, ctx.Writer.Status()))
		}
	}
}

// timeout middleware wraps the request context with a timeout
func (r *rest) SetTimeout(ctx *gin.Context) {
	// wrap the request context with a timeout
	c, cancel := context.WithTimeout(ctx.Request.Context(), r.conf.Timeout)

	// cancel to clear resources after finished
	defer cancel()

	c = appcontext.SetRequestStartTime(c, time.Now())

	// replace request with context wrapped request
	ctx.Request = ctx.Request.WithContext(c)
	ctx.Next()

}

func (r *rest) addFieldsToContext(ctx *gin.Context) {
	reqid := ctx.GetHeader(header.KeyRequestID)
	if reqid == "" {
		reqid = uuid.New().String()
	}

	c := ctx.Request.Context()
	c = appcontext.SetRequestId(c, reqid)
	c = appcontext.SetUserAgent(c, ctx.Request.Header.Get(header.KeyUserAgent))
	c = appcontext.SetAcceptLanguage(c, ctx.Request.Header.Get(header.KeyAcceptLanguage))
	c = appcontext.SetServiceVersion(c, r.conf.Meta.Version)
	c = appcontext.SetDeviceType(c, ctx.Request.Header.Get(header.KeyDeviceType))
	c = appcontext.SetCacheControl(c, ctx.Request.Header.Get(header.KeyCacheControl))
	ctx.Request = ctx.Request.WithContext(c)
	ctx.Next()
}

func (r *rest) httpRespError(ctx *gin.Context, err error) {
	c := ctx.Request.Context()

	if errors.Is(c.Err(), context.DeadlineExceeded) {
		err = errors.NewWithCode(codes.CodeContextDeadlineExceeded, "%s", "Context Deadline Exceeded")
	}

	httpStatus, displayError := errors.Compile(err, appcontext.GetAcceptLanguage(ctx))
	statusStr := http.StatusText(httpStatus)

	errResp := &entity.HTTPResp{
		Message: entity.HTTPMessage{
			Title: displayError.Title,
			Body:  displayError.Body,
		},
		Meta: entity.Meta{
			Path:       r.conf.Meta.Host + ctx.Request.URL.String(),
			StatusCode: httpStatus,
			Status:     statusStr,
			Message:    fmt.Sprintf("%s %s [%d] %s", ctx.Request.Method, ctx.Request.URL.RequestURI(), httpStatus, statusStr),
			Error: &entity.MetaError{
				Code:    int(displayError.Code),
				Message: err.Error(),
			},
			Timestamp: time.Now().Format(time.RFC3339),
			RequestID: appcontext.GetRequestId(c),
		},
	}

	r.log.Error(c, err)

	c = appcontext.SetAppResponseCode(c, displayError.Code)
	c = appcontext.SetAppErrorMessage(c, fmt.Sprintf("%s - %s", displayError.Title, displayError.Body))
	c = appcontext.SetResponseHttpCode(c, httpStatus)
	ctx.Request = ctx.Request.WithContext(c)

	ctx.Header(header.KeyRequestID, appcontext.GetRequestId(c))
	ctx.AbortWithStatusJSON(httpStatus, errResp)
}

func (r *rest) httpRespSuccess(ctx *gin.Context, code codes.Code, data interface{}, p *entity.Pagination) {
	successApp := codes.Compile(code, appcontext.GetAcceptLanguage(ctx))
	c := ctx.Request.Context()
	meta := entity.Meta{
		Path:       r.conf.Meta.Host + ctx.Request.URL.String(),
		StatusCode: successApp.StatusCode,
		Status:     http.StatusText(successApp.StatusCode),
		Message:    fmt.Sprintf("%s %s [%d] %s", ctx.Request.Method, ctx.Request.URL.RequestURI(), successApp.StatusCode, http.StatusText(successApp.StatusCode)),
		Timestamp:  time.Now().Format(time.RFC3339),
		RequestID:  appcontext.GetRequestId(c),
	}

	resp := &entity.HTTPResp{
		Message: entity.HTTPMessage{
			Title: successApp.Title,
			Body:  successApp.Body,
		},
		Meta:       meta,
		Data:       data,
		Pagination: p,
	}

	reqstart := appcontext.GetRequestStartTime(c)
	if !time.Time.IsZero(reqstart) {
		resp.Meta.TimeElapsed = fmt.Sprintf("%dms", int64(time.Since(reqstart)/time.Millisecond))
	}

	raw, err := r.json.Marshal(&resp)
	if err != nil {
		r.httpRespError(ctx, errors.NewWithCode(codes.CodeInternalServerError, "%s", err.Error()))
		return
	}

	c = appcontext.SetAppResponseCode(c, code)
	c = appcontext.SetResponseHttpCode(c, successApp.StatusCode)
	ctx.Request = ctx.Request.WithContext(c)

	ctx.Header(header.KeyRequestID, appcontext.GetRequestId(c))
	ctx.Data(successApp.StatusCode, header.ContentTypeJSON, raw)
}

// Bind request body to struct using tag 'json'
func (r *rest) Bind(ctx *gin.Context, obj interface{}) error {
	err := ctx.ShouldBindWith(obj, binding.Default(ctx.Request.Method, ctx.ContentType()))
	if err != nil {
		return errors.NewWithCode(codes.CodeBadRequest, "%s", err.Error())
	}

	return nil
}

// Bind all query params to struct using tag 'form'
func (r *rest) BindQuery(ctx *gin.Context, obj interface{}) error {
	err := ctx.ShouldBindWith(obj, binding.Query)
	if err != nil {
		return errors.NewWithCode(codes.CodeBadRequest, "%s", err.Error())
	}

	return nil
}

// Bind uri params to struct using tag 'uri'
func (r *rest) BindUri(ctx *gin.Context, obj interface{}) error {
	err := ctx.ShouldBindUri(obj)
	if err != nil {
		return errors.NewWithCode(codes.CodeBadRequest, "%s", err.Error())
	}

	return nil
}

// Bind all params (query and uri params) to struct using tag 'uri' and 'form'
func (r *rest) BindParams(ctx *gin.Context, obj interface{}) error {
	err := r.BindQuery(ctx, obj)
	if err != nil {
		return errors.NewWithCode(codes.CodeBadRequest, "%s", err.Error())
	}

	err = r.BindUri(ctx, obj)
	if err != nil {
		return errors.NewWithCode(codes.CodeBadRequest, "%s", err.Error())
	}

	return nil
}

// ReadRequestBytesFromContext read body request from context in bytes
func (r *rest) ReadRequestBytesFromContext(ctx *gin.Context) []byte {
	var buffer bytes.Buffer
	teeReader := io.TeeReader(ctx.Request.Body, &buffer)
	body, _ := io.ReadAll(teeReader)

	return body
}

// @Summary Health Check
// @Description This endpoint will hit the server
// @Tags Server
// @Produce json
// @Success 200 string example="PONG!"
// @Router /ping [GET]
func (r *rest) Ping(ctx *gin.Context) {
	r.httpRespSuccess(ctx, codes.CodeSuccess, "PONG!", nil)
}
