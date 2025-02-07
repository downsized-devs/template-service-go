package usecase

import (
	"github.com/downsized-devs/sdk-go/auth"
	"github.com/downsized-devs/sdk-go/logger"
	"github.com/downsized-devs/sdk-go/parser"
	"github.com/downsized-devs/template-service-go/src/business/domain"
)

type Usecases struct {
	// Add usecase package interfaces here
}

type InitParam struct {
	Log    logger.Interface
	Parser parser.Parser
	Dom    *domain.Domains
	Auth   auth.Interface
}

func Init(param InitParam) *Usecases {
	dom := &Usecases{}

	return dom
}
