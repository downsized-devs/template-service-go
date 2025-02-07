package domain

import (
	"net/http"

	"github.com/downsized-devs/sdk-go/logger"
	"github.com/downsized-devs/sdk-go/parser"
	"github.com/downsized-devs/sdk-go/sql"
)

type Domains struct {
	// Add domain package interfaces here
}

type InitParam struct {
	Log    logger.Interface
	Db     sql.Interface
	Parser parser.Parser
	Http   *http.Client
}

func Init(param InitParam) *Domains {
	dom := &Domains{}

	return dom
}
