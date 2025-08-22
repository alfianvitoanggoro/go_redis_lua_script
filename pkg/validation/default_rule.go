package validation

import (
	"errors"
	"strings"
)

// GetDefaultAmountPointByCurrencyCode mengembalikan amount_point default berdasarkan currency_code
func GetDefaultAmountPointByCurrencyCode(currencyCode string) (float64, error) {
	currency := strings.ToUpper(currencyCode)

	switch currency {
	case "USD":
		return 20, nil
	case "SGD":
		return 20, nil
	default:
		return 0, errors.New("unsupported currency")
	}
}
