package main

import (
	"fmt"
	"os"

	"remnawave-tg-shop-bot/internal/moynalog"
)

func main() {
	// Получите логин и пароль для аутентификации из переменных окружения
	username := os.Getenv("MOYNALOG_USERNAME")
	password := os.Getenv("MOYNALOG_PASSWORD")
	if username == "" || password == "" {
		fmt.Println("Ошибка: переменные окружения MOYNALOG_USERNAME и MOYNALOG_PASSWORD должны быть установлены")
		os.Exit(1)
	}

	// Создайте новый клиент moynalog с помощью moynalog.NewClient()
	client := moynalog.NewClient()

	// Вызовите метод Authenticate() клиента moynalog, передав полученные логин и пароль
	authResp, err := client.Authenticate(username, password)
	if err != nil {
		// В случае ошибки выведите сообщение об ошибке
		fmt.Printf("Ошибка аутентификации: %v\n", err)
		os.Exit(1)
	}

	// В случае успешной аутентификации выведите полученный токен в консоль
	fmt.Printf("Токен аутентификации: %s\n", authResp.Token)
}
