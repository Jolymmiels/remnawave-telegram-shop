package utils

import (
	"fmt"
	"remnawave-tg-shop-bot/internal/translation"
	"strconv"
	"strings"
	"time"
)

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

func FormatDateByLanguage(date time.Time, langCode string) string {
	tm := translation.GetInstance()

	monthKeys := []string{
		"month_january", "month_february", "month_march", "month_april",
		"month_may", "month_june", "month_july", "month_august",
		"month_september", "month_october", "month_november", "month_december",
	}

	monthName := tm.GetText(langCode, monthKeys[date.Month()-1])

	if langCode == "ru" {
		return fmt.Sprintf("%d %s %d (%02d:%02d)",
			date.Day(), monthName, date.Year(), date.Hour(), date.Minute())
	}

	return fmt.Sprintf("%s %d, %d (%02d:%02d)",
		monthName, date.Day(), date.Year(), date.Hour(), date.Minute())
}
