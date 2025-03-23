package config

import (
	"os"
	"strconv"
)

func Price12() int {
	price, err := strconv.Atoi(os.Getenv("PRICE_12"))
	if err != nil {
		return 1200 // Значение по умолчанию
	}
	return price
} 