# Планирование задач в команде GophKeeper

## 📋 Оглавление
1. [Общая организация](#общая-организация)
2. [Еженедельное планирование](#еженедельное-планирование)
3. [Git Workflow](#git-workflow)
4. [Задачи и приоритеты](#задачи-и-приоритеты)
5. [Code Review Process](#code-review-process)
6. [Шаблоны и чеклисты](#шаблоны-и-чеклисты)
7. [Коммуникация](#коммуникация)

---

## Общая организация

### Состав команды

| Роль | Ответственность | Фокус |
|------|----------------|-------|
| **Developer 1** | Backend Lead | gRPC Server, Database, Services, Auth |
| **Developer 2** | Client Lead | CLI, TUI, gRPC Client, Crypto |

**Общие задачи:**
- Code review друг друга
- Архитектурные решения
- Integration тесты
- Документация

---

## Еженедельное планирование

### Sprint Structure (2-недельные спринты)

```
Sprint 1 (Week 1-2): Foundation & Auth
Sprint 2 (Week 3-4): CRUD Operations
Sprint 3 (Week 5-6): Sync & Polish
```

---

## 📅 Sprint 1: Foundation & Auth (Week 1-2)

### Week 1: Infrastructure Setup

#### 🔵 Developer 1: Backend Infrastructure

**Задачи:**

```markdown
## Task 1.1: Project Structure & Dependencies
**Priority:** 🔴 Critical
**Estimate:** 4 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Создать структуру директорий (cmd, internal, pkg, api, migrations)
- [ ] Инициализировать go.mod
- [ ] Установить зависимости (viper, grpc, pgx, goose, jwt, zap)
- [ ] Создать Makefile с базовыми командами
- [ ] Создать .gitignore

### Acceptance Criteria:
- Структура проекта соответствует TECH_STACK.md
- `make build` успешно компилирует проект
- Все зависимости установлены

### Branch: `feature/backend-structure`
```

```markdown
## Task 1.2: Proto Files & Code Generation
**Priority:** 🔴 Critical
**Estimate:** 6 hours
**Status:** 📋 Todo
**Depends on:** Task 1.1

### Checklist:
- [ ] Установить buf
- [ ] Создать buf.yaml, buf.gen.yaml
- [ ] Написать auth.proto (Register, Login, Refresh, Logout)
- [ ] Написать credentials.proto (CRUD для логин/пароль)
- [ ] Написать texts.proto (CRUD для текстов)
- [ ] Написать cards.proto (CRUD для карт)
- [ ] Написать binaries.proto (CRUD для файлов)
- [ ] Написать sync.proto (синхронизация)
- [ ] Сгенерировать Go код через buf
- [ ] Добавить make target `generate-proto`

### Acceptance Criteria:
- Все proto файлы компилируются без ошибок
- Код генерируется в gen/go/
- `make generate-proto` работает

### Branch: `feature/proto-definitions`

### Review Points:
⚠️ Обязательно ревью с Dev 2 перед генерацией!
```

```markdown
## Task 1.3: Database Migrations
**Priority:** 🔴 Critical
**Estimate:** 4 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Создать миграцию 00001_create_users.sql
- [ ] Создать миграцию 00002_create_credentials.sql
- [ ] Создать миграцию 00003_create_texts.sql
- [ ] Создать миграцию 00004_create_cards.sql
- [ ] Создать миграцию 00005_create_binaries.sql
- [ ] Создать миграцию 00006_create_refresh_tokens.sql
- [ ] Добавить индексы
- [ ] Протестировать up/down миграции
- [ ] Добавить make targets для миграций

### Acceptance Criteria:
- Все миграции применяются без ошибок
- Down миграции корректно откатывают изменения
- Индексы созданы на внешних ключах и часто используемых полях

### Branch: `feature/database-migrations`
```

```markdown
## Task 1.4: Docker Compose & Config
**Priority:** 🟠 High
**Estimate:** 3 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Создать docker-compose.yml (PostgreSQL + MinIO)
- [ ] Создать configs/server.yaml
- [ ] Создать Dockerfile.server (multi-stage build)
- [ ] Реализовать config loader с Viper
- [ ] Реализовать logger wrapper (Zap)
- [ ] Добавить health check endpoints

### Acceptance Criteria:
- `make docker-up` запускает PostgreSQL и MinIO
- Сервер читает конфигурацию из yaml
- Логирование работает (structured logs)

### Branch: `feature/docker-and-config`
```

```markdown
## Task 1.5: PostgreSQL Repository Layer
**Priority:** 🔴 Critical
**Estimate:** 4 hours
**Status:** 📋 Todo
**Depends on:** Task 1.3

### Checklist:
- [ ] Создать DB connection с pgxpool
- [ ] Реализовать Users Repository (Create, GetByUsername, GetByID)
- [ ] Реализовать health check для БД
- [ ] Добавить connection pooling настройки
- [ ] Написать unit тесты (моки)

### Acceptance Criteria:
- Connection pool настроен (min/max connections)
- Все методы репозитория протестированы
- Graceful shutdown работает

### Branch: `feature/repository-layer`
```

---

#### 🟢 Developer 2: Client Infrastructure

```markdown
## Task 2.1: Client Project Structure
**Priority:** 🔴 Critical
**Estimate:** 3 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Создать структуру client директорий
- [ ] Инициализировать client dependencies (cobra, bubbletea, viper, keyring)
- [ ] Создать cmd/client/main.go
- [ ] Настроить Cobra root command
- [ ] Добавить version command с build info
- [ ] Создать Makefile targets для client

### Acceptance Criteria:
- `make build-client` компилирует клиент
- `gophkeeper --help` показывает справку
- `gophkeeper version` показывает версию

### Branch: `feature/client-structure`
```

```markdown
## Task 2.2: gRPC Client Wrapper
**Priority:** 🔴 Critical
**Estimate:** 5 hours
**Status:** 📋 Todo
**Depends on:** Task 1.2 (proto)
**Blocked by:** Dev 1 proto generation

### Checklist:
- [ ] Создать gRPC client connection
- [ ] Реализовать API client wrapper
- [ ] Добавить auth context (JWT в metadata)
- [ ] Реализовать retry logic
- [ ] Реализовать timeout handling
- [ ] Добавить error mapping (gRPC -> user-friendly)

### Acceptance Criteria:
- Client подключается к серверу
- JWT токен добавляется в metadata
- Ошибки обрабатываются корректно

### Branch: `feature/grpc-client`

### Notes:
⚠️ Можно начать с моками, если proto еще не готовы
```

```markdown
## Task 2.3: Configuration & Keyring
**Priority:** 🔴 Critical
**Estimate:** 4 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Реализовать config loader (Viper + XDG)
- [ ] Создать ~/.gophkeeper/config.yaml структуру
- [ ] Реализовать Keyring wrapper для токенов
- [ ] Добавить SaveToken/LoadToken/DeleteToken
- [ ] Fallback на encrypted file если keyring недоступен
- [ ] Реализовать config commands (set/get)

### Acceptance Criteria:
- Конфиг читается из ~/.gophkeeper/config.yaml
- Токены безопасно хранятся в системном хранилище
- На macOS использует Keychain
- На Linux использует Secret Service
- На Windows использует Credential Manager

### Branch: `feature/client-config`
```

```markdown
## Task 2.4: Client-Side Encryption
**Priority:** 🔴 Critical
**Estimate:** 5 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Реализовать key derivation (Scrypt from master password)
- [ ] Реализовать AES-256-GCM encryption
- [ ] Реализовать AES-256-GCM decryption
- [ ] Реализовать salt generation
- [ ] Добавить helper функции для string encryption
- [ ] Написать unit тесты (encrypt/decrypt cycle)

### Acceptance Criteria:
- Шифрование/дешифрование работает корректно
- Каждое шифрование использует уникальный nonce
- Тесты покрывают edge cases
- Производительность приемлемая (< 10ms для строки)

### Branch: `feature/client-crypto`
```

```markdown
## Task 2.5: CLI Base Commands
**Priority:** 🟠 High
**Estimate:** 3 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Создать структуру команд (auth, cred, text, card, file, sync)
- [ ] Реализовать глобальные флаги (--config, --server, --debug)
- [ ] Добавить color output (fatih/color)
- [ ] Реализовать table rendering для списков
- [ ] Добавить progress bar для long operations
- [ ] Реализовать interactive input (password prompts)

### Acceptance Criteria:
- Все команды зарегистрированы в cobra
- Help текст информативный
- Output красиво форматирован

### Branch: `feature/cli-base`
```

---

#### 📊 Week 1 Summary

**Developer 1 Total:** ~21 hours (5 tasks)  
**Developer 2 Total:** ~20 hours (5 tasks)

**Weekly Sync Points:**
- **Day 2:** Review proto файлов вместе
- **Day 4:** Standup - проверка прогресса
- **Day 5:** Integration check - client может подключиться к серверу

---

### Week 2: Authentication Implementation

#### 🔵 Developer 1: Auth Service

```markdown
## Task 1.6: Auth Service Implementation
**Priority:** 🔴 Critical
**Estimate:** 8 hours
**Status:** 📋 Todo
**Depends on:** Task 1.5

### Checklist:
- [ ] Реализовать password hashing (Argon2)
- [ ] Реализовать password verification
- [ ] Реализовать JWT token generation (access + refresh)
- [ ] Реализовать JWT token validation
- [ ] Реализовать refresh token storage (DB)
- [ ] Реализовать Register logic
- [ ] Реализовать Login logic
- [ ] Реализовать Logout logic (invalidate tokens)
- [ ] Написать unit тесты (покрытие > 80%)

### Acceptance Criteria:
- Пароли хешируются с Argon2
- JWT токены подписываются и валидируются
- Refresh token работает корректно
- Все методы покрыты тестами

### Branch: `feature/auth-service`

### Test Cases:
- ✅ Регистрация нового пользователя
- ✅ Регистрация с существующим username (должна упасть)
- ✅ Логин с правильными credentials
- ✅ Логин с неправильным паролем
- ✅ Логин несуществующего пользователя
- ✅ Refresh токена
- ✅ Refresh с невалидным токеном
- ✅ Logout
```

```markdown
## Task 1.7: gRPC Auth Handlers
**Priority:** 🔴 Critical
**Estimate:** 4 hours
**Status:** 📋 Todo
**Depends on:** Task 1.6

### Checklist:
- [ ] Реализовать Register handler
- [ ] Реализовать Login handler
- [ ] Реализовать RefreshToken handler
- [ ] Реализовать Logout handler
- [ ] Добавить input validation
- [ ] Добавить error handling
- [ ] Написать integration тесты

### Acceptance Criteria:
- Все handlers возвращают корректные proto responses
- Ошибки мапятся в gRPC status codes
- Validation работает на всех inputs

### Branch: `feature/grpc-auth-handlers`
```

```markdown
## Task 1.8: gRPC Middleware
**Priority:** 🔴 Critical
**Estimate:** 5 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Реализовать Auth Interceptor (JWT validation)
- [ ] Реализовать Logging Interceptor
- [ ] Реализовать Recovery Interceptor (panic handling)
- [ ] Реализовать Rate Limiting Interceptor
- [ ] Настроить middleware chain
- [ ] Добавить whitelisted методы (Register, Login)
- [ ] Написать тесты для interceptors

### Acceptance Criteria:
- JWT проверяется на каждом запросе (кроме whitelisted)
- Все запросы логируются
- Panic не роняет сервер
- Rate limiting защищает от abuse

### Branch: `feature/grpc-middleware`
```

```markdown
## Task 1.9: Credentials Repository
**Priority:** 🟠 High
**Estimate:** 4 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Реализовать Create credential
- [ ] Реализовать List credentials by user
- [ ] Реализовать Get credential by ID
- [ ] Реализовать Update credential
- [ ] Реализовать Delete credential (soft delete)
- [ ] Добавить фильтрацию и пагинацию
- [ ] Написать unit тесты

### Acceptance Criteria:
- Все CRUD операции работают
- Soft delete используется вместо hard delete
- Pagination корректно работает
- Тесты покрывают все методы

### Branch: `feature/credentials-repository`
```

---

#### 🟢 Developer 2: Auth Commands

```markdown
## Task 2.6: Register Command
**Priority:** 🔴 Critical
**Estimate:** 4 hours
**Status:** 📋 Todo
**Depends on:** Task 2.2, 2.3
**Blocked by:** Task 1.7 (server auth handlers)

### Checklist:
- [ ] Реализовать `gophkeeper register` command
- [ ] Добавить interactive prompts (username, password, confirm password)
- [ ] Добавить password strength validation
- [ ] Реализовать вызов gRPC Register
- [ ] Добавить error handling
- [ ] Написать unit тесты

### Acceptance Criteria:
- Команда успешно регистрирует пользователя
- Пароль не виден при вводе
- Password confirmation работает
- Ошибки показываются user-friendly

### Branch: `feature/register-command`

### CLI Example:
```bash
$ gophkeeper register
Username: john
Password: ********
Confirm password: ********
✓ User registered successfully!
```
```

```markdown
## Task 2.7: Login Command
**Priority:** 🔴 Critical
**Estimate:** 5 hours
**Status:** 📋 Todo
**Depends on:** Task 2.6

### Checklist:
- [ ] Реализовать `gophkeeper login` command
- [ ] Добавить interactive prompts
- [ ] Реализовать вызов gRPC Login
- [ ] Сохранить токены в keyring
- [ ] Сохранить текущего пользователя в config
- [ ] Добавить флаги --username, --password (для скриптов)
- [ ] Написать unit тесты

### Acceptance Criteria:
- Успешный логин сохраняет токены
- Токены используются в последующих запросах
- Можно логиниться с флагами (non-interactive)
- Ошибки обрабатываются корректно

### Branch: `feature/login-command`

### CLI Example:
```bash
$ gophkeeper login
Username: john
Password: ********
✓ Logged in successfully!

# Or non-interactive
$ gophkeeper login --username john --password secret
✓ Logged in successfully!
```
```

```markdown
## Task 2.8: Logout Command
**Priority:** 🟠 High
**Estimate:** 2 hours
**Status:** 📋 Todo
**Depends on:** Task 2.7

### Checklist:
- [ ] Реализовать `gophkeeper logout` command
- [ ] Вызвать gRPC Logout на сервере
- [ ] Удалить токены из keyring
- [ ] Очистить текущего пользователя в config
- [ ] Добавить confirmation prompt
- [ ] Написать unit тесты

### Acceptance Criteria:
- Токены удаляются из keyring
- После logout команды требуют auth
- Confirmation можно пропустить с флагом --force

### Branch: `feature/logout-command`
```

```markdown
## Task 2.9: Credentials CLI - Add/List
**Priority:** 🟠 High
**Estimate:** 6 hours
**Status:** 📋 Todo
**Depends on:** Task 2.7, Task 2.4 (crypto)

### Checklist:
- [ ] Реализовать `gophkeeper cred add` command
- [ ] Добавить interactive prompts (name, login, password, url, metadata)
- [ ] Реализовать client-side encryption перед отправкой
- [ ] Реализовать `gophkeeper cred list` command
- [ ] Добавить table rendering для списка
- [ ] Добавить пагинацию
- [ ] Добавить search flag
- [ ] Написать unit тесты

### Acceptance Criteria:
- Credentials шифруются на клиенте
- Список отображается в красивой таблице
- Поиск работает
- Пагинация работает

### Branch: `feature/credentials-cli-basic`

### CLI Example:
```bash
$ gophkeeper cred add
Name: GitHub Account
Login: john@example.com
Password: ********
URL: https://github.com
Metadata (optional): Personal account
✓ Credential saved successfully!

$ gophkeeper cred list
┌────────────────────────┬──────────────────┬─────────────────────┐
│ Name                   │ Login            │ URL                 │
├────────────────────────┼──────────────────┼─────────────────────┤
│ GitHub Account         │ john@example.com │ https://github.com  │
│ Gmail                  │ john@gmail.com   │ https://gmail.com   │
└────────────────────────┴──────────────────┴─────────────────────┘

$ gophkeeper cred list --search github
┌────────────────────────┬──────────────────┬─────────────────────┐
│ Name                   │ Login            │ URL                 │
├────────────────────────┼──────────────────┼─────────────────────┤
│ GitHub Account         │ john@example.com │ https://github.com  │
└────────────────────────┴──────────────────┴─────────────────────┘
```
```

```markdown
## Task 2.10: Integration Testing Helper
**Priority:** 🟢 Medium
**Estimate:** 3 hours
**Status:** 📋 Todo

### Checklist:
- [ ] Создать test helper для запуска тестового сервера
- [ ] Создать test fixtures (пользователи, credentials)
- [ ] Реализовать cleanup между тестами
- [ ] Написать пример integration теста (register -> login -> add cred)

### Acceptance Criteria:
- Integration тесты легко писать
- Тесты изолированы друг от друга
- Cleanup работает корректно

### Branch: `feature/integration-test-helpers`
```

---

#### 📊 Week 2 Summary

**Developer 1 Total:** ~21 hours (4 tasks)  
**Developer 2 Total:** ~20 hours (5 tasks)

**Weekly Sync Points:**
- **Day 2:** Auth service готов - начинаем интеграцию
- **Day 3:** Совместный integration test (register -> login)
- **Day 5:** Sprint review - демо auth flow

**Integration Test (Together):**
```bash
# Dev 1: Запускает сервер
make run-server

# Dev 2: Тестирует клиент
gophkeeper register
gophkeeper login
gophkeeper cred add
gophkeeper cred list
```

---

## 📅 Sprint 2: CRUD Operations (Week 3-4)

### Week 3: Credentials & Texts

#### Task Distribution

| Day | Developer 1 (Backend) | Developer 2 (Client) |
|-----|-----------------------|----------------------|
| Mon | Credentials Service | Review proto, Planning |
| Tue | Credentials gRPC handlers | Credentials CLI (get/update/delete) |
| Wed | Texts Service | Texts CLI (add/list) |
| Thu | Texts gRPC handlers | Texts CLI (get/update/delete) |
| Fri | Integration tests | Integration tests, Bug fixes |

---

### Week 4: Cards & Binaries

#### Task Distribution

| Day | Developer 1 (Backend) | Developer 2 (Client) |
|-----|-----------------------|----------------------|
| Mon | Cards Service | Cards CLI (add/list) |
| Tue | Cards gRPC handlers | Cards CLI (get/update/delete) |
| Wed | MinIO setup | Files CLI (upload) |
| Thu | Binaries Service | Files CLI (download/list) |
| Fri | Integration tests | Integration tests |

---

## Git Workflow

### Branch Naming Convention

```
feature/backend-auth-service
feature/client-login-command
bugfix/jwt-expiration
hotfix/security-vulnerability
docs/api-documentation
test/integration-auth-flow
```

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: новая фича
- `fix`: баг фикс
- `docs`: документация
- `test`: добавление тестов
- `refactor`: рефакторинг
- `perf`: улучшение производительности
- `chore`: рутинные задачи (зависимости, конфиг)

**Examples:**

```bash
feat(auth): implement JWT token generation

- Add access token generation with 1h expiry
- Add refresh token with 30d expiry
- Add token validation middleware

Closes #12
```

```bash
fix(client): handle connection timeout gracefully

Previously, client would panic on timeout.
Now shows user-friendly error message.

Fixes #45
```

```bash
test(server): add integration tests for auth flow

- Test register -> login -> refresh
- Test invalid credentials
- Test token expiration

Related to #23
```

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review performed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] No new warnings
- [ ] Coverage not decreased

## Testing
How to test these changes:
1. Step 1
2. Step 2
3. Expected result

## Screenshots (if applicable)

## Related Issues
Closes #XX
Related to #YY
```

### Code Review Checklist

**For Reviewer:**

```markdown
## Functionality
- [ ] Code does what it's supposed to do
- [ ] Edge cases handled
- [ ] Error handling is appropriate

## Code Quality
- [ ] Code is readable and maintainable
- [ ] No unnecessary complexity
- [ ] Functions are small and focused
- [ ] No code duplication

## Testing
- [ ] Tests are present
- [ ] Tests are meaningful
- [ ] Coverage is adequate
- [ ] Edge cases tested

## Security
- [ ] No hardcoded secrets
- [ ] Input validation present
- [ ] No SQL injection vulnerabilities
- [ ] Authentication/authorization checked

## Performance
- [ ] No obvious performance issues
- [ ] Database queries optimized
- [ ] No N+1 queries

## Documentation
- [ ] Code is self-documenting
- [ ] Complex logic explained
- [ ] Public functions documented
- [ ] README updated if needed
```

---

## Задачи и приоритеты

### Priority Levels

🔴 **Critical (P0)** - Блокирует других
- Должно быть сделано в первую очередь
- Блокирует других разработчиков
- Критично для MVP

🟠 **High (P1)** - Важно для спринта
- Должно быть в текущем спринте
- Важно для функциональности
- Видимо пользователям

🟡 **Medium (P2)** - Можно отложить
- Nice to have в текущем спринте
- Можно перенести в следующий
- Улучшения UX

🟢 **Low (P3)** - Бэклог
- Не критично
- Оптимизации
- Рефакторинг

### Task States

```
📋 Todo          - Задача в бэклоге
🎯 This Week     - Запланирована на эту неделю
🔨 In Progress   - В работе
🔄 In Review     - На code review
✅ Done          - Завершено
🐛 Bug           - Найден баг, нужен фикс
⏸️ Blocked       - Заблокирована другой задачей
❌ Cancelled     - Отменена
```

### GitHub Projects Setup

**Columns:**

```
┌──────────────┬──────────────┬──────────────┬──────────────┬──────────────┐
│  📋 Backlog  │ 🎯 This Week │ 🔨 In Progress│  🔄 Review   │   ✅ Done    │
├──────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│              │              │              │              │              │
│  Task 3.1    │  Task 1.1    │  Task 1.2    │  Task 1.5    │  Task 0.1    │
│  Task 3.2    │  Task 1.3    │  Task 2.1    │              │  Task 0.2    │
│  Task 3.3    │  Task 1.4    │              │              │  Task 0.3    │
│              │  Task 2.2    │              │              │              │
│              │  Task 2.3    │              │              │              │
│              │              │              │              │              │
└──────────────┴──────────────┴──────────────┴──────────────┴──────────────┘
```

**Rules:**
- Максимум 2 задачи "In Progress" на человека
- PR должен быть в "Review" максимум 1 день
- "This Week" планируется в понедельник

---

## Коммуникация

### Daily Standup (15 минут, 09:00)

**Формат:**

```markdown
## Developer 1 (Backend)

### ✅ Вчера:
- Завершил Task 1.2: Proto файлы
- Начал Task 1.3: Миграции БД

### 🎯 Сегодня:
- Закончу Task 1.3: Миграции БД
- Начну Task 1.4: Docker compose

### 🚧 Блокеры:
- Нет

### 💬 Вопросы:
- Нужно обсудить proto для sync (встреча 15 минут после standup?)
```

```markdown
## Developer 2 (Client)

### ✅ Вчера:
- Завершил Task 2.1: Client structure
- Изучил proto файлы от Dev 1

### 🎯 Сегодня:
- Task 2.2: gRPC Client wrapper
- Task 2.3: Config & Keyring

### 🚧 Блокеры:
- Жду proto генерации для полной интеграции (но работаю с моками)

### 💬 Вопросы:
- Да, давай обсудим sync proto
```

---

### Weekly Planning (Понедельник, 30 минут)

**Agenda:**

```markdown
## Sprint X, Week Y Planning

### 1. Review прошлой недели (10 мин)
- Что завершили?
- Что не успели? Почему?
- Сколько story points закрыли?

### 2. Планирование на неделю (15 мин)
- Какие задачи берем в This Week?
- Есть ли зависимости?
- Realisticли план?

### 3. Блокеры и риски (5 мин)
- Что может пойти не так?
- Нужна ли помощь извне?
```

**Example Planning:**

```markdown
## Sprint 1, Week 2 Planning - Mon, Jan 15

### Last Week Review
✅ Completed:
- Task 1.1-1.5: Backend infrastructure (100%)
- Task 2.1-2.4: Client infrastructure (100%)

❌ Not completed:
- Task 2.5: CLI Base (80% done, moving to this week)

📊 Velocity: 19 story points / 20 planned (95%)

### This Week Goals
🎯 Focus: Authentication End-to-End

Developer 1:
- Task 1.6: Auth Service (8h) - 🔴 P0
- Task 1.7: gRPC Auth Handlers (4h) - 🔴 P0
- Task 1.8: gRPC Middleware (5h) - 🔴 P0
- Task 1.9: Credentials Repo (4h) - 🟠 P1
Total: 21h

Developer 2:
- Task 2.5: CLI Base (1h remaining) - 🟠 P1
- Task 2.6: Register Command (4h) - 🔴 P0
- Task 2.7: Login Command (5h) - 🔴 P0
- Task 2.8: Logout Command (2h) - 🟠 P1
- Task 2.9: Credentials CLI (6h) - 🟠 P1
- Task 2.10: Integration Tests (3h) - 🟢 P2
Total: 21h

### Sync Points
- Wednesday: Auth integration test together
- Friday: Sprint review demo

### Risks
⚠️ Auth middleware может занять больше времени
Mitigation: Dev 1 начинает с этого во вторник

⚠️ Dev 2 ждет server auth handlers
Mitigation: Dev 2 работает с моками параллельно
```

---

### Weekly Retrospective (Пятница, 30 минут)

**Format:**

```markdown
## Sprint X, Week Y Retrospective - Fri, Jan 19

### What went well? 👍
- Proto files review процесс сработал отлично
- Daily standups помогли синхронизироваться
- Integration test прошел с первого раза

### What didn't go well? 👎
- Task 1.8 занял 7 часов вместо 5
- Code review занял 2 дня для Task 2.6
- Docker compose падал из-за версии PostgreSQL

### What to improve? 💡
- Лучше оценивать сложные middleware задачи
- Делать PR меньше (< 300 lines)
- Документировать Docker issues

### Action Items 🎯
- [ ] @dev1: Создать troubleshooting guide для Docker
- [ ] @both: Стремиться к PR < 300 lines
- [ ] @both: Code review в течение 4 часов max
```

---

### Async Communication (Slack/Telegram)

**Channels:**

```
#gophkeeper-general    - Общие вопросы
#gophkeeper-backend    - Backend обсуждения
#gophkeeper-client     - Client обсуждения
#gophkeeper-prs        - PR notifications
#gophkeeper-ci         - CI/CD notifications
```

**Communication Guidelines:**

```markdown
✅ DO:
- Используйте threads для обсуждений
- @mention когда нужен ответ
- Используйте code snippets для кода
- Добавляйте контекст в вопросы

❌ DON'T:
- Не пишите "Привет" и ждите ответа - сразу пишите вопрос
- Не используйте @channel без крайней необходимости
- Не дублируйте в несколько каналов
```

---

## Шаблоны и чеклисты

### Template: New Feature Task

```markdown
## Task X.Y: [Feature Name]
**Priority:** 🔴/🟠/🟡/🟢
**Estimate:** X hours
**Status:** 📋 Todo
**Assignee:** @username
**Sprint:** Sprint X, Week Y
**Depends on:** Task A, Task B
**Blocks:** Task C

### Description
[What needs to be done]

### Technical Details
[How it should be implemented]

### Checklist:
- [ ] Implementation
- [ ] Unit tests (coverage > 80%)
- [ ] Integration tests (if applicable)
- [ ] Documentation (godoc)
- [ ] Code review passed
- [ ] CI green

### Acceptance Criteria:
- [ ] Criterion 1
- [ ] Criterion 2
- [ ] Criterion 3

### Test Cases:
- ✅ Test case 1
- ✅ Test case 2
- ✅ Test case 3

### Branch: `feature/[name]`

### Related Issues:
Closes #XX
Related to #YY

### Notes:
[Any additional context]
```

---

### Template: Bug Report

```markdown
## 🐛 Bug: [Short Description]
**Priority:** 🔴/🟠/🟡/🟢
**Status:** 📋 Todo
**Assignee:** @username
**Found in:** Sprint X, Week Y

### Description
[What's wrong]

### Steps to Reproduce
1. Step 1
2. Step 2
3. Step 3

### Expected Behavior
[What should happen]

### Actual Behavior
[What actually happens]

### Environment
- OS: [macOS/Linux/Windows]
- Version: [v1.0.0]
- Go version: [1.21]

### Logs/Screenshots
```
[paste logs]
```

### Possible Solution
[If you have an idea]

### Branch: `bugfix/[name]`

### Related Issues:
Fixes #XX
```

---

### Template: Code Review Request

```markdown
## 🔍 Ready for Review: [PR Title]

Hi @reviewer! 👋

This PR implements [feature/fix description].

### Changes:
- Change 1
- Change 2
- Change 3

### How to Test:
```bash
# Step 1
make docker-up

# Step 2
make run-server

# Step 3
gophkeeper login
gophkeeper cred add
```

### Screenshots:
[if UI changes]

### Checklist:
✅ All tests pass
✅ Coverage > 80%
✅ Linter happy
✅ Documentation updated

### Questions/Concerns:
- Question about [specific part]
- Not sure about [implementation detail]

### Files to Pay Attention To:
- `internal/server/service/auth.go` - main logic
- `internal/server/service/auth_test.go` - test coverage

Thanks! 🙏
```

---

### Checklist: Before Pushing

```markdown
## Pre-Push Checklist

- [ ] Code compiles without errors
- [ ] All tests pass locally
- [ ] Linter passes (make lint)
- [ ] No debugging code left (console.logs, etc)
- [ ] No commented out code
- [ ] No TODOs without issue links
- [ ] Commit messages are clear
- [ ] Branch is up to date with main
```

---

### Checklist: Before Merging PR

```markdown
## Pre-Merge Checklist

- [ ] Code review approved
- [ ] All CI checks pass
- [ ] No merge conflicts
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Changelog updated (if needed)
- [ ] No new linter warnings
- [ ] Coverage not decreased
```

---

## Tools & Setup

### Recommended VS Code Extensions

```json
{
  "recommendations": [
    "golang.go",
    "github.copilot",
    "eamodio.gitlens",
    "ms-azuretools.vscode-docker",
    "bufbuild.vscode-buf",
    "esbenp.prettier-vscode"
  ]
}
```

### Git Hooks (pre-commit)

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running pre-commit checks..."

# Run tests
echo "Running tests..."
go test ./... || exit 1

# Run linter
echo "Running linter..."
golangci-lint run ./... || exit 1

# Check formatting
echo "Checking formatting..."
gofmt -l . | grep -v vendor | grep . && exit 1

echo "✅ All checks passed!"
exit 0
```

---

## Metrics & Tracking

### Weekly Metrics to Track

```markdown
## Week X Metrics

### Velocity
- Story points completed: 19/20
- Tasks completed: 8/10
- Bugs found: 3
- Bugs fixed: 2

### Code Quality
- Test coverage: 78%
- Linter warnings: 0
- Code review time (avg): 6 hours

### Process
- PR size (avg): 280 lines
- Time to review (avg): 4 hours
- Time to merge (avg): 8 hours

### Team
- Standup attendance: 100%
- Blocked days: 0.5 days
- Pair programming: 2 sessions
```

---

## Example: First Week in Action

### Monday, Jan 15

**09:00 - Standup**
```
Dev 1: Starting with project structure
Dev 2: Starting with client structure
No blockers
```

**10:00 - 12:00**
- Dev 1: Task 1.1 (Project structure)
- Dev 2: Task 2.1 (Client structure)

**14:00 - 18:00**
- Dev 1: Task 1.2 (Proto files) - WIP
- Dev 2: Task 2.1 (Client structure) - Complete ✅

**18:00 - Git**
```bash
# Dev 2 pushes first PR
git push origin feature/client-structure
# Opens PR, requests review from Dev 1
```

---

### Tuesday, Jan 16

**09:00 - Standup**
```
Dev 1: Finished proto files, need review with Dev 2
Dev 2: Completed client structure, ready for gRPC client
Sync point: 10:00 - proto review (30 min)
```

**10:00 - Proto Review Meeting (Both)**
- Review all proto definitions
- Discuss field types
- Agree on error codes
- Dev 1 makes changes if needed

**11:00 - 18:00**
- Dev 1: Generate proto code, start Task 1.3 (Migrations)
- Dev 2: Code review for Dev 1's proto PR, start Task 2.2 (gRPC client)

---

### Wednesday, Jan 17

**09:00 - Standup**
```
Dev 1: Migrations 80% done, will finish today
Dev 2: gRPC client wrapper 60% done
No blockers
```

**Full day coding**
- Dev 1: Complete migrations, start Docker compose
- Dev 2: Complete gRPC client, start config

**Evening**
- Dev 1: PR for migrations
- Dev 2: PR for gRPC client

---

### Thursday, Jan 18

**Code Reviews Morning**
- Dev 1 reviews Dev 2's PR
- Dev 2 reviews Dev 1's PR
- Both merge after approval

**Coding continues...**

---

### Friday, Jan 19

**Demo Time!**
```bash
# Dev 1 starts server
make docker-up
make run-server

# Dev 2 tests client
gophkeeper --help
gophkeeper version
```

**Retrospective at 16:00**

**Planning for next week**

---

## 🎯 Success Criteria

Команда работает хорошо, если:

✅ **Communication**
- Daily standups занимают ровно 15 минут
- Blockers решаются в течение дня
- Вопросы не висят без ответа > 4 часов

✅ **Process**
- Все знают, что делать каждый день
- PR ревьюится в течение 4-8 часов
- Нет сюрпризов в конце спринта

✅ **Quality**
- Coverage > 70% всегда
- Linter не ругается
- CI зеленый 95%+ времени

✅ **Team**
- Нет работы в выходные
- Нет переработок
- Positive atmosphere

---

## Appendix: Tools List

### Must Have
- **Git** - version control
- **GitHub** - repository hosting
- **GitHub Projects** - task management
- **Slack/Telegram** - communication
- **VS Code** - IDE

### Nice to Have
- **Notion** - documentation
- **Excalidraw** - diagrams
- **Postman** - API testing
- **Docker Desktop** - containerization

---

**Good Luck! 🚀**

*Remember: Communication > Code*
