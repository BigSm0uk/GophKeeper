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
	masterPassword := "my-super-secret-password"

	fmt.Println("🔄 Тест синхронизации")
	fmt.Println("=====================")
	fmt.Println()

	sm, err := storage.InitializeStorage(dbPath, username, masterPassword)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}
	defer sm.Close()

	// 1. Показать все данные
	fmt.Println("📊 Текущее состояние хранилища:")
	fmt.Println("--------------------------------")

	creds, _ := sm.Encrypted.ListCredentials()
	cards, _ := sm.Encrypted.ListCards()
	texts, _ := sm.Encrypted.ListTexts()
	binaries, _ := sm.Encrypted.ListBinaries()

	fmt.Printf("\n✅ Credentials: %d\n", len(creds))
	for i, cred := range creds {
		fmt.Printf("   %d. %s (login: %s) [%s]\n", i+1, cred.Name, cred.Login, cred.SyncStatus)
	}

	fmt.Printf("\n💳 Cards: %d\n", len(cards))
	for i, card := range cards {
		fmt.Printf("   %d. %s (**** %s) [%s]\n", i+1, card.Name, card.CardNumber[len(card.CardNumber)-4:], card.SyncStatus)
	}

	fmt.Printf("\n📝 Texts: %d\n", len(texts))
	for i, text := range texts {
		fmt.Printf("   %d. %s [%s]\n", i+1, text.Name, text.SyncStatus)
	}

	fmt.Printf("\n📁 Binaries: %d\n", len(binaries))
	for i, binary := range binaries {
		fmt.Printf("   %d. %s (%s) [%s]\n", i+1, binary.Name, binary.Filename, binary.SyncStatus)
	}

	// 2. Проверить несинхронизированные
	fmt.Println("\n\n🔄 Проверка несинхронизированных данных:")
	fmt.Println("-----------------------------------------")

	pending, _ := sm.Encrypted.GetPendingCredentials()
	fmt.Printf("Credentials ожидающих синхронизации: %d\n", len(pending))

	if len(pending) > 0 {
		fmt.Println("\n📤 Симуляция отправки на сервер:")
		for _, cred := range pending {
			fmt.Printf("   Отправка '%s'...\n", cred.Name)
			// Симуляция отправки
			err := sm.Encrypted.UpdateCredentialSyncStatus(cred.ID, storage.StatusSynced)
			if err != nil {
				fmt.Printf("   ❌ Ошибка: %v\n", err)
			} else {
				fmt.Printf("   ✅ Успешно синхронизировано\n")
			}
		}
	}

	// 3. Добавить новый credential
	fmt.Println("\n\n➕ Добавление нового credential:")
	fmt.Println("--------------------------------")

	newCred := storage.CreateCredentialWithEncryption(
		"gitlab-1",
		"GitLab Account",
		"user@gitlab.com",
		"secret-password",
		nil,
		nil,
	)

	err = sm.Encrypted.SaveCredential(newCred)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
	} else {
		fmt.Println("✅ Новый credential сохранен")
		fmt.Printf("   Статус: %s (ожидает синхронизации)\n", newCred.SyncStatus)
	}

	// 4. Финальная статистика
	fmt.Println("\n\n📊 Финальная статистика:")
	fmt.Println("------------------------")

	pending2, _ := sm.Encrypted.GetPendingCredentials()
	fmt.Printf("Credentials всего: %d\n", len(creds)+1)
	fmt.Printf("Ожидают синхронизации: %d\n", len(pending2))

	fmt.Println("\n✅ Тест синхронизации завершен успешно!")
}
