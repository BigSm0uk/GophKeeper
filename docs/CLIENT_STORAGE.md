# Клиентское хранилище с шифрованием

## 📦 Что реализовано

### 1. Локальная база данных (SQLite)

**Файл:** `internal/client/storage/local_db.go`

Таблицы:
- ✅ `credentials` - логины/пароли
- ✅ `cards` - банковские карты  
- ✅ `texts` - текстовые заметки
- ✅ `binaries` - метаданные бинарных файлов
- ✅ `sync_queue` - очередь синхронизации

### 2. Шифрование (AES-256-GCM)

**Файл:** `internal/client/crypto/crypto.go`

- ✅ Клиентское E2E шифрование
- ✅ AES-256-GCM алгоритм
- ✅ Scrypt для деривации ключа (N=32768, r=8, p=1)
- ✅ Уникальный salt для каждого пользователя
- ✅ Методы `Encrypt()` и `Decrypt()`

### 3. Безопасное хранилище токенов (Keyring)

**Файл:** `internal/client/storage/keyring.go`

- ✅ Интеграция с системным keyring:
  - macOS: Keychain
  - Linux: Secret Service API
  - Windows: Credential Manager
- ✅ Хранение access/refresh токенов
- ✅ Хранение salt для шифрования
- ✅ Хранение хеша мастер-пароля
- ✅ Методы для работы с текущим пользователем

### 4. Автоматическое шифрование данных

**Файл:** `internal/client/storage/encrypted_storage.go`

Обертка над LocalDB с автоматическим шифрованием/дешифрованием:
- ✅ `SaveCredential()` / `GetCredential()` / `ListCredentials()` / `DeleteCredential()`
- ✅ `SaveCard()` / `GetCard()` / `ListCards()` / `DeleteCard()`
- ✅ `SaveText()` / `GetText()` / `ListTexts()` / `DeleteText()`
- ✅ `SaveBinary()` / `GetBinary()` / `ListBinaries()` / `DeleteBinary()`
- ✅ Управление статусами синхронизации

### 5. Менеджер хранилища (StorageManager)

**Файл:** `internal/client/storage/factory.go`

- ✅ Фабрика для инициализации всего стека
- ✅ Автоматическая генерация salt для новых пользователей
- ✅ Верификация мастер-пароля для существующих пользователей
- ✅ Метод смены мастер-пароля с перешифрованием
- ✅ Полная очистка данных пользователя

### 6. Вспомогательные файлы

**Файлы:**
- ✅ `internal/client/storage/status.go` - статусы синхронизации
- ✅ `internal/client/storage/cards_texts.go` - методы для cards и texts
- ✅ `internal/client/storage/binaries.go` - методы для binaries
- ✅ `internal/client/storage/README.md` - полная документация
- ✅ `internal/client/storage/integration_test.go` - интеграционные тесты
- ✅ `internal/client/storage/example/main.go` - пример использования

## 🔐 Архитектура шифрования

```
Мастер-пароль пользователя
         ↓
    Scrypt (N=32768, r=8, p=1) + уникальный salt
         ↓
    256-битный ключ шифрования
         ↓
    AES-256-GCM (для каждого поля свой nonce)
         ↓
    Зашифрованные данные → SQLite
```

### Что шифруется?

**Credentials:**
- ✅ Login, Password, URL, Metadata
- ❌ ID, Name (для поиска/отображения)

**Cards:**
- ✅ CardNumber, ExpiryDate, CVV, Metadata
- ❌ ID, Name, CardholderName, BankName

**Texts:**
- ✅ Content, Metadata
- ❌ ID, Name

**Binaries:**
- ✅ Metadata
- ❌ ID, Name, Filename, Size, ContentType, Checksum

## 📋 Использование

### Базовый пример

```go
import "github.com/BigSm0uk/GophKeeper/internal/client/storage"

// Инициализация
sm, err := storage.InitializeStorage(
    "./data/local.db",
    "user@example.com",
    "master-password",
)
defer sm.Close()

// Создание и сохранение credential (автоматически шифруется)
cred := storage.CreateCredentialWithEncryption(
    "github-1", "GitHub", "login", "password", nil, nil,
)
sm.Encrypted.SaveCredential(cred)

// Получение (автоматически расшифровывается)
retrieved, _ := sm.Encrypted.GetCredential("github-1")
fmt.Println(retrieved.Login) // "login"
```

Полная документация: `internal/client/storage/README.md`

## 🧪 Тестирование

```bash
# Запуск всех тестов
go test ./internal/client/storage/... -v

# Запуск основного примера
go run ./internal/client/storage/example/demo/main.go

# Запуск теста шифрования (быстрый, без keyring)
go run ./internal/client/storage/example/test-crypto/test_crypto.go
```

**Внимание:** Тесты используют системный keyring и могут запрашивать разрешения на macOS.

## 🔄 Статусы синхронизации

```go
const (
    StatusSynced    = "synced"    // Синхронизировано
    StatusPending   = "pending"   // Ожидает отправки
    StatusUploading = "uploading" // В процессе
    StatusUpdated   = "updated"   // Изменено локально
)
```

## 🎯 Интеграция с TUI

Для использования в TUI нужно:

1. При запуске клиента инициализировать `StorageManager`:
```go
sm, err := storage.InitializeStorage(dbPath, username, masterPassword)
```

2. Передать `sm.Encrypted` в TUI компоненты для работы с данными

3. Использовать `sm.TokenStore` для управления токенами

4. При выходе вызвать `sm.Close()`

## 🔒 Безопасность

✅ **Реализовано:**
- Клиентское E2E шифрование
- Zero-knowledge архитектура (сервер не видит расшифрованные данные)
- Проверенные алгоритмы (AES-256-GCM, Scrypt)
- Токены в системном keyring
- Уникальный salt для каждого пользователя
- Хеш мастер-пароля для верификации

⚠️ **Важно:**
- Потеря мастер-пароля = потеря всех локальных данных
- Нет механизма восстановления пароля
- При смене пароля требуется перешифрование

## 📊 Структура файлов

```
internal/client/storage/
├── local_db.go              # SQLite база данных
├── keyring.go               # Системный keyring
├── encrypted_storage.go     # Автоматическое шифрование
├── factory.go               # StorageManager
├── crypto.go                # AES-256-GCM (уже был)
├── status.go                # Статусы синхронизации
├── cards_texts.go           # Методы для cards/texts
├── binaries.go              # Методы для binaries
├── README.md                # Документация
├── CHECKLIST.md             # Чек-лист
├── HOW_TO_TEST.md           # Инструкции по тестированию
├── integration_test.go      # Интеграционные тесты
├── keyring_test.go          # Тесты keyring
├── local_db_test.go         # Тесты БД
└── example/
    ├── demo/main.go                    # Основной пример
    ├── test-crypto/test_crypto.go      # Тест шифрования
    ├── test-password/test_password.go  # Тест паролей
    └── test-sync/test_sync.go          # Тест синхронизации
```

## 🚀 Следующие шаги

Для полной интеграции нужно:

1. **TUI интеграция:**
   - Добавить форму ввода мастер-пароля при первом запуске
   - Интегрировать `StorageManager` в TUI flow
   - Реализовать экраны для CRUD операций с данными

2. **Синхронизация с сервером:**
   - Использовать `GetPendingCredentials()` для получения несинхронизированных данных
   - Отправлять зашифрованные данные на сервер через gRPC
   - Обновлять статусы через `UpdateCredentialSyncStatus()`
   - Реализовать conflict resolution (по времени изменения)

3. **Binaries:**
   - Реализовать шифрование самих файлов на диске
   - Реализовать chunked upload/download

4. **Дополнительно:**
   - Автоматическая синхронизация в фоне
   - Экспорт/импорт данных
   - Метрики и логирование

## ✅ Готово к использованию

Все компоненты локального хранилища с шифрованием реализованы и протестированы. Можно приступать к интеграции с TUI и синхронизацией с сервером.
