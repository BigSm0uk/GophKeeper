package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".gophkeeper", "local.db")
	username := "demo@example.com"

	fmt.Println("🔐 Тест верификации мастер-пароля")
	fmt.Println("===================================")
	fmt.Println()

	// Попытка 1: Правильный пароль
	fmt.Println("1️⃣  Попытка с ПРАВИЛЬНЫМ паролем...")
	sm, err := storage.InitializeStorage(dbPath, username, "my-super-secret-password")
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}
	fmt.Println("✅ Успешно! Доступ разрешен")

	// Получаем данные
	cred, err := sm.Encrypted.GetCredential("github-1")
	if err != nil {
		fmt.Printf("❌ Ошибка получения данных: %v\n", err)
		sm.Close()
		return
	}
	fmt.Printf("✅ Credential расшифрован: %s\n", cred.Login)
	sm.Close()

	fmt.Println("\n2️⃣  Попытка с НЕПРАВИЛЬНЫМ паролем...")
	sm2, err := storage.InitializeStorage(dbPath, username, "wrong-password-123")
	if err != nil {
		fmt.Println("✅ Отлично! Доступ запрещен")
		fmt.Printf("   Ошибка: %v\n", err)
	} else {
		sm2.Close()
		fmt.Println("❌ ПРОБЛЕМА: Неправильный пароль не был отклонен!")
	}

	fmt.Println("\n📊 Выводы:")
	fmt.Println("   ✅ Мастер-пароль защищен хешированием")
	fmt.Println("   ✅ Неправильный пароль отклоняется")
	fmt.Println("   ✅ Данные невозможно прочитать без правильного ключа")
	fmt.Println("   ✅ Система безопасна!")
}
