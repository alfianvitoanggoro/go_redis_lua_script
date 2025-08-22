package handler

import (
	"grls/internal/modules/wallet/dto"
	"grls/internal/modules/wallet/usecase"
	"grls/pkg/logger"
	"grls/pkg/response"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

type WalletHandler struct {
	usecase *usecase.WalletUsecase
}

func NewWalletHandler(u *usecase.WalletUsecase) *WalletHandler {
	return &WalletHandler{usecase: u}
}

func (h *WalletHandler) Deposit(c *fiber.Ctx) error {
	var req dto.DepositInput
	if err := c.BodyParser(&req); err != nil {
		errMsg := err.Error()
		logger.WriteLogToFile("failed", "WalletHandler.Deposit.Parser", req, &errMsg)
		return response.WriteError(c, fiber.StatusBadRequest, "Invalid request body", errMsg)
	}
	if err := validate.Struct(&req); err != nil {
		errMsg := err.Error()
		logger.WriteLogToFile("failed", "WalletHandler.Deposit.Validate", req, &errMsg)
		return response.WriteError(c, fiber.StatusBadRequest, "Validation error", errMsg)
	}

	out, err := h.usecase.Deposit(c.Context(), req)
	if err != nil {
		errMsg := err.Error()
		logger.WriteLogToFile("failed", "WalletHandler.Deposit.Usecase", req, &errMsg)
		// gunakan code dari usecase untuk hint error client
		return response.WriteError(c, fiber.StatusInternalServerError, "Failed to process deposit", errMsg)
	}

	logger.WriteLogToFile("success", "WalletHandler.Deposit", req, nil)
	return response.WriteSuccess(c, fiber.StatusCreated, "Deposit processed", out)
}
