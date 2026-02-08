package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
)

func main() {
	// Настройка
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	dbPath := filepath.Join(homeDir, ".gophkeeper", "local.db")
	username := "demo@example.com"
	masterPassword := "my-super-secret-password"

	fmt.Println("🔐 GophKeeper Storage Demo")
	fmt.Println("===========================")
	fmt.Printf("Database: %s\n", dbPath)
	fmt.Printf("Username: %s\n\n", username)

	// Инициализация хранилища
	fmt.Println("📦 Initializing storage...")
	sm, err := storage.InitializeStorage(dbPath, username, masterPassword, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer sm.Close()

	fmt.Println("✅ Storage initialized successfully!")
	fmt.Println()

	// Демонстрация Credentials
	fmt.Println("🔑 Working with Credentials")
	fmt.Println("----------------------------")

	// Создать credential
	cred := storage.CreateCredentialWithEncryption(
		"github-1",
		"GitHub Account",
		"john.doe@example.com",
		"very-secret-password",
		stringPtr("https://github.com"),
		stringPtr(`{"note": "Work account", "2fa": "enabled"}`),
	)

	fmt.Println("Saving credential...")
	if err := sm.Encrypted.SaveCredential(cred); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Credential saved (encrypted in database)")

	// Получить credential
	fmt.Println("\nRetrieving credential...")
	retrieved, err := sm.Encrypted.GetCredential("github-1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✅ Credential retrieved (decrypted):\n")
	fmt.Printf("   Name:     %s\n", retrieved.Name)
	fmt.Printf("   Login:    %s\n", retrieved.Login)
	fmt.Printf("   Password: %s\n", retrieved.Password)
	fmt.Printf("   URL:      %s\n", *retrieved.URL)
	fmt.Printf("   Metadata: %s\n", *retrieved.Metadata)

	// Список credentials
	fmt.Println("\nListing all credentials...")
	creds, err := sm.Encrypted.ListCredentials()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Found %d credential(s)\n", len(creds))

	// Демонстрация Cards
	fmt.Println("\n💳 Working with Cards")
	fmt.Println("---------------------")

	card := storage.CreateCardWithEncryption(
		"visa-1",
		"Work Visa",
		"4111111111111111",
		"John Doe",
		"12/25",
		"123",
		stringPtr("Chase Bank"),
		nil,
	)

	fmt.Println("Saving card...")
	if err := sm.Encrypted.SaveCard(card); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Card saved (encrypted)")

	retrievedCard, err := sm.Encrypted.GetCard("visa-1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✅ Card retrieved (decrypted):\n")
	fmt.Printf("   Name:       %s\n", retrievedCard.Name)
	fmt.Printf("   Number:     %s\n", maskCardNumber(retrievedCard.CardNumber))
	fmt.Printf("   Holder:     %s\n", retrievedCard.CardholderName)
	fmt.Printf("   Expiry:     %s\n", retrievedCard.ExpiryDate)
	fmt.Printf("   CVV:        ***\n")

	// Демонстрация Texts
	fmt.Println("\n📝 Working with Texts")
	fmt.Println("---------------------")

	text := storage.CreateTextWithEncryption(
		"note-1",
		"Meeting Notes",
		"Discussion about Q4 roadmap. Key points:\n- Feature X launch in November\n- Budget approval pending\n- Hiring 2 new developers",
		stringPtr(`{"tags": ["important", "meeting", "roadmap"]}`),
	)

	fmt.Println("Saving text note...")
	if err := sm.Encrypted.SaveText(text); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Text saved (encrypted)")

	retrievedText, err := sm.Encrypted.GetText("note-1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✅ Text retrieved (decrypted):\n")
	fmt.Printf("   Name:    %s\n", retrievedText.Name)
	fmt.Printf("   Content: %s\n", truncate(retrievedText.Content, 50))

	// Демонстрация Binaries
	fmt.Println("\n📁 Working with Binaries")
	fmt.Println("------------------------")

	binary := storage.CreateBinaryWithEncryption(
		"doc-1",
		"Passport Scan",
		"passport.pdf",
		"/tmp/passport.pdf",
		1024000,
		"application/pdf",
		"abc123def456789",
		stringPtr(`{"encrypted": true, "category": "documents"}`),
	)

	fmt.Println("Saving binary metadata...")
	if err := sm.Encrypted.SaveBinary(binary); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Binary metadata saved")

	retrievedBinary, err := sm.Encrypted.GetBinary("doc-1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✅ Binary metadata retrieved:\n")
	fmt.Printf("   Name:     %s\n", retrievedBinary.Name)
	fmt.Printf("   Filename: %s\n", retrievedBinary.Filename)
	fmt.Printf("   Size:     %d bytes\n", retrievedBinary.Size)
	fmt.Printf("   Type:     %s\n", retrievedBinary.ContentType)

	// Демонстрация Sync Status
	fmt.Println("\n🔄 Sync Status")
	fmt.Println("---------------")

	pending, err := sm.Encrypted.GetPendingCredentials()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Items pending sync: %d\n", len(pending))

	// Помечаем как синхронизированное
	if len(pending) > 0 {
		fmt.Printf("Marking '%s' as synced...\n", pending[0].Name)
		err = sm.Encrypted.UpdateCredentialSyncStatus(pending[0].ID, storage.StatusSynced)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("✅ Status updated")
	}

	// Работа с токенами
	fmt.Println("\n🎫 Token Management")
	fmt.Println("-------------------")

	fmt.Println("Saving access and refresh tokens...")
	if err := sm.TokenStore.SaveAccessToken(username, "eyJhbGc..."); err != nil {
		log.Fatal(err)
	}
	if err := sm.TokenStore.SaveRefreshToken(username, "refresh_xyz..."); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Tokens saved to system keyring")

	accessToken, err := sm.TokenStore.GetAccessToken(username)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Access token retrieved: %s\n", truncate(accessToken, 20))

	// Итоги
	fmt.Println("\n📊 Summary")
	fmt.Println("----------")
	allCreds, _ := sm.Encrypted.ListCredentials()
	allCards, _ := sm.Encrypted.ListCards()
	allTexts, _ := sm.Encrypted.ListTexts()
	allBinaries, _ := sm.Encrypted.ListBinaries()

	fmt.Printf("Credentials: %d\n", len(allCreds))
	fmt.Printf("Cards:       %d\n", len(allCards))
	fmt.Printf("Texts:       %d\n", len(allTexts))
	fmt.Printf("Binaries:    %d\n", len(allBinaries))

	fmt.Println("\n✅ Demo completed successfully!")
	fmt.Println("\n🔒 All data is encrypted in the local database using AES-256-GCM")
	fmt.Println("🔑 Master password is derived using Scrypt (N=32768, r=8, p=1)")
	fmt.Println("🗝️  Tokens are stored in system keyring (Keychain/Secret Service/Credential Manager)")
}

func stringPtr(s string) *string {
	return &s
}

func maskCardNumber(number string) string {
	if len(number) <= 4 {
		return "****"
	}
	return "****-****-****-" + number[len(number)-4:]
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}
