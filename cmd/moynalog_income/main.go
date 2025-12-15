package main

import (
	"fmt"
	"os"
	"strconv"

	"remnawave-tg-shop-bot/internal/moynalog"
)

func main() {
	// Получите токен аутентификации из переменной окружения MOYNALOG_AUTH_TOKEN
	token := os.Getenv("MOYNALOG_AUTH_TOKEN")

	if token == "" {
		fmt.Println("Ошибка: переменная окружения MOYNALOG_AUTH_TOKEN должна быть установлена")
		os.Exit(1)
	}

	// Получите amount (сумма) и comment (комментарий) из аргументов командной строки
	if len(os.Args) < 3 {
		fmt.Println("Ошибка: необходимо указать сумму и комментарий в качестве аргументов")
		fmt.Println("Использование: go run main.go <сумма> \"<комментарий>\"")
		os.Exit(1)
	}

	amountStr := os.Args[1]
	comment := os.Args[2]

	// Преобразуйте строку суммы в число
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		fmt.Printf("Ошибка: невозможно преобразовать сумму '%s' в число\n", amountStr)
		os.Exit(1)
	}

	// Создайте новый клиент moynalog с помощью moynalog.NewClient()
	client := moynalog.NewClient()

	// Вызовите метод CreateIncome() клиента moynalog, передав полученный токен, сумму и комментарий
	incomeResp, err := client.CreateIncome(token, amount, comment)
	if err != nil {
		// В случае ошибки выведите сообщение об ошибке
		fmt.Printf("Ошибка при создании дохода: %v\n", err)
		os.Exit(1)
	}

	// В случае успешного создания дохода выведите полученный ответ в консоль
	fmt.Printf("Доход успешно создан: %+v\n", incomeResp)
}
