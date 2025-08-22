package factory

import (
	"grls/internal/infrastructure/repository"
	"grls/internal/modules/wallet/handler"
	"grls/internal/modules/wallet/store"
	"grls/internal/modules/wallet/usecase"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func newWalletFactory(dbWrite *gorm.DB, dbRead *gorm.DB, rdb redis.UniversalClient) *handler.WalletHandler {
	store := store.NewRedisWalletStore(rdb)
	walletRepo := repository.NewWalletRepository(dbWrite, dbRead)
	usecase := usecase.NewWalletUsecase(walletRepo, store)
	handler := handler.NewWalletHandler(usecase)
	return handler
}
