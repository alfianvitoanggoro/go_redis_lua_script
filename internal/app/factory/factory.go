package factory

import (
	"grls/internal/infrastructure/repository"

	"gorm.io/gorm"
)

type Factory struct {
	WalletRepository *repository.WalletRepository
}

func NewFactory(dbWrite *gorm.DB, dbRead *gorm.DB) *Factory {
	return &Factory{
		WalletRepository: repository.NewWalletRepository(dbWrite, dbRead),
	}
}
