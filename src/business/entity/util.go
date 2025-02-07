package entity

import (
	"github.com/downsized-devs/sdk-go/null"
)

const (
	SystemID   int64  = -1
	SystemName string = "system"
)

type UtilityColumn struct {
	Status    int
	Flag      int
	Meta      string
	CreatedAt null.Time
	CreatedBy string
	UpdatedAt null.Time
	UpdatedBy string
	DeletedAt null.Time
	DeletedBy string
}
