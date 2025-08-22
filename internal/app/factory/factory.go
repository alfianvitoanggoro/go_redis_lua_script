package factory

import (
	point_handler "grls/internal/modules/wallet/handler"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	WalletHandler *point_handler.WalletHandler
}

func Build(dbWrite *gorm.DB, dbRead *gorm.DB, rdb redis.UniversalClient) *Container {
	return &Container{
		WalletHandler: newWalletFactory(dbWrite, dbRead, rdb),
	}
}
