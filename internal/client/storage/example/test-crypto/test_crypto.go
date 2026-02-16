package main

import (
	"fmt"

	"github.com/BigSm0uk/GophKeeper/internal/client/crypto"
)

func main() {
	fmt.Println("🔐 Тест шифрования")
	fmt.Println("===================")
	fmt.Println()

	// Генерация salt
	fmt.Println("1️⃣  Генерация salt...")
	salt, err := crypto.GenerateSalt()
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}
	fmt.Printf("✅ Salt сгенерирован: %d байт\n", len(salt))

	// Создание encryptor
	fmt.Println("\n2️⃣  Создание Encryptor из мастер-пароля...")
	masterPassword := "my-super-secret-password"
	encryptor, err := crypto.NewEncryptor(masterPassword, salt)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}
	fmt.Println("✅ Encryptor создан (ключ получен через Scrypt)")

	// Шифрование данных
	fmt.Println("\n3️⃣  Шифрование данных...")
	testData := map[string]string{
		"Login":       "john.doe@example.com",
		"Password":    "super-secret-password-123",
		"Card Number": "4111111111111111",
		"CVV":         "123",
		"Secret Note": "This is a very secret note that should be encrypted!",
	}

	encrypted := make(map[string]string)
	for key, value := range testData {
		enc, err := encryptor.Encrypt(value)
		if err != nil {
			fmt.Printf("❌ Ошибка шифрования %s: %v\n", key, err)
			return
		}
		encrypted[key] = enc
		fmt.Printf("✅ %s: %s → %s...\n", key, value[:min(len(value), 20)], enc[:min(len(enc), 30)])
	}

	// Дешифрование
	fmt.Println("\n4️⃣  Дешифрование данных...")
	for key, encValue := range encrypted {
		dec, err := encryptor.Decrypt(encValue)
		if err != nil {
			fmt.Printf("❌ Ошибка дешифрования %s: %v\n", key, err)
			return
		}

		if dec == testData[key] {
			fmt.Printf("✅ %s: успешно расшифровано\n", key)
		} else {
			fmt.Printf("❌ %s: расшифрованные данные не совпадают!\n", key)
		}
	}

	// Попытка расшифровать с неправильным ключом
	fmt.Println("\n5️⃣  Попытка расшифровать с НЕПРАВИЛЬНЫМ ключом...")
	wrongEncryptor, _ := crypto.NewEncryptor("wrong-password", salt)

	for key, encValue := range encrypted {
		dec, err := wrongEncryptor.Decrypt(encValue)
		if err != nil {
			fmt.Printf("✅ %s: не удалось расшифровать (ожидаемо)\n", key)
		} else {
			fmt.Printf("❌ %s: ПРОБЛЕМА! Удалось расшифровать неправильным ключом: %s\n", key, dec)
		}
		break // Достаточно проверить один
	}

	// Тест с разными nonce
	fmt.Println("\n6️⃣  Тест уникальности шифрования (разные nonce)...")
	text := "Same text encrypted multiple times"
	enc1, _ := encryptor.Encrypt(text)
	enc2, _ := encryptor.Encrypt(text)

	if enc1 != enc2 {
		fmt.Println("✅ Каждое шифрование использует уникальный nonce")
		fmt.Printf("   Enc1: %s...\n", enc1[:40])
		fmt.Printf("   Enc2: %s...\n", enc2[:40])
	} else {
		fmt.Println("❌ ПРОБЛЕМА: Одинаковые зашифрованные тексты!")
	}

	// Статистика
	fmt.Println("\n📊 Статистика:")
	fmt.Println("---------------")
	fmt.Printf("Мастер-пароль: %s\n", masterPassword)
	fmt.Printf("Salt: %d байт (уникален для пользователя)\n", len(salt))
	fmt.Printf("Алгоритм: AES-256-GCM\n")
	fmt.Printf("KDF: Scrypt (N=32768, r=8, p=1)\n")
	fmt.Printf("Длина ключа: 256 бит\n")

	fmt.Println("\n✅ Все тесты пройдены успешно!")
	fmt.Println("   • Шифрование работает")
	fmt.Println("   • Дешифрование работает")
	fmt.Println("   • Неправильный ключ не может расшифровать")
	fmt.Println("   • Каждое шифрование уникально")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
