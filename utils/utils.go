package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func FormatSubscriptionDescription(months int) string {
	var unit string
	switch months {
	case 1:
		unit = "месяц"
	case 2, 3, 4:
		unit = "месяца"
	default:
		unit = "месяцев"
	}
	return fmt.Sprintf("Подписка на %d %s", months, unit)
}

func MaskHalfInt(input int) string {
	return MaskHalf(strconv.Itoa(input))
}

func MaskHalfInt64(input int64) string {
	return MaskHalf(strconv.FormatInt(input, 10))
}

func MaskHalf(input string) string {
	if input == "" {
		return input
	}
	if len(input) < 2 {
		return input
	}
	length := len(input)
	visibleLength := length / 2
	maskedLength := length - visibleLength
	return input[:visibleLength] + strings.Repeat("*", maskedLength)
}
