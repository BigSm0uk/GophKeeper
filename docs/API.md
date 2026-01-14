# API Endpoints - GophKeeper

## Описание API

Документация описывает REST API для клиент-серверной системы GophKeeper - менеджера паролей.

Базовый URL: `https://api.gophkeeper.example.com/api/v1`

## Аутентификация

Все защищенные эндпоинты используют **OAuth 2.0 (Bearer Token)**:
```
Authorization: Bearer <access_token>
```

Используем поток **Resource Owner Password Credentials (ROPC)** для первого релиза (CLI/desktop-клиент доверенный, без сторонних приложений). Refresh токены обновляют access токены. В будущем можно добавить Authorization Code + PKCE без изменения остальной схемы.

---

## Endpoints

### 1. Аутентификация и авторизация

#### 1.1. Регистрация пользователя
```
POST /auth/register
```

**Request Body:**
```json
{
  "username": "string",
  "password": "string",
  "email": "string" (optional)
}
```

**Response:** `201 Created`
```json
{
  "user_id": "uuid",
  "username": "string",
  "created_at": "timestamp"
}
```

**Errors:**
- `400` - Невалидные данные
- `409` - Пользователь уже существует

---

#### 1.2. Получение токена (OAuth2 Password Grant)
```
POST /oauth/token
```

**Request Body (grant_type=password):**
```json
{
  "grant_type": "password",
  "username": "string",
  "password": "string",
  "client_id": "gophkeeper-cli",
  "client_secret": "string (optional for public clients)",
  "scope": "openid offline_access" 
}
```

**Response:** `200 OK`
```json
{
  "access_token": "string",
  "refresh_token": "string",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid offline_access"
}
```

**Errors:**
- `400` - Неверный grant_type / отсутствуют параметры
- `401` - Неверные учетные данные или client_id
- `403` - Клиент не имеет права на парольный грант

---

#### 1.3. Обновление токена (Refresh Token Grant)
```
POST /oauth/token
```

**Request Body (grant_type=refresh_token):**
```json
{
  "grant_type": "refresh_token",
  "refresh_token": "string",
  "client_id": "gophkeeper-cli",
  "client_secret": "string (optional for public clients)"
}
```

**Response:** `200 OK`
```json
{
  "access_token": "string",
  "refresh_token": "string",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid offline_access"
}
```

**Errors:**
- `400` - Неверный grant_type / отсутствуют параметры
- `401` - Невалидный или отозванный refresh token

---

#### 1.4. Отзыв токена (Logout / Revoke)
```
POST /oauth/revoke
```

**Headers:** Authorization: Basic base64(client_id:client_secret) или в теле

**Request Body:**
```json
{
  "token": "string",              // refresh_token или access_token
  "token_type_hint": "refresh_token|access_token"
}
```

**Response:** `200 OK`
```json
{
  "revoked": true
}
```

**Errors:**
- `400` - Отсутствует token
- `401` - Неверные client credentials

---

### 2. Управление данными логин/пароль

#### 2.1. Создание записи логин/пароль
```
POST /credentials
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "name": "string",
  "login": "string",
  "password": "string",
  "url": "string" (optional),
  "metadata": "string" (optional)
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "name": "string",
  "login": "string",
  "password": "string (encrypted)",
  "url": "string",
  "metadata": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### 2.2. Получение всех записей логин/пароль
```
GET /credentials
```

**Headers:** Authorization required

**Query Parameters:**
- `limit` (optional) - количество записей
- `offset` (optional) - смещение для пагинации
- `search` (optional) - поиск по имени

**Response:** `200 OK`
```json
{
  "items": [
    {
      "id": "uuid",
      "name": "string",
      "login": "string",
      "url": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ],
  "total": 100,
  "limit": 20,
  "offset": 0
}
```

---

#### 2.3. Получение конкретной записи логин/пароль
```
GET /credentials/{id}
```

**Headers:** Authorization required

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "name": "string",
  "login": "string",
  "password": "string (encrypted)",
  "url": "string",
  "metadata": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

**Errors:**
- `404` - Запись не найдена

---

#### 2.4. Обновление записи логин/пароль
```
PUT /credentials/{id}
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "name": "string",
  "login": "string",
  "password": "string",
  "url": "string",
  "metadata": "string"
}
```

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "name": "string",
  "login": "string",
  "password": "string (encrypted)",
  "url": "string",
  "metadata": "string",
  "updated_at": "timestamp"
}
```

---

#### 2.5. Удаление записи логин/пароль
```
DELETE /credentials/{id}
```

**Headers:** Authorization required

**Response:** `204 No Content`

**Errors:**
- `404` - Запись не найдена

---

### 3. Управление текстовыми данными

#### 3.1. Создание текстовой записи
```
POST /texts
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "name": "string",
  "content": "string",
  "metadata": "string" (optional)
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "name": "string",
  "content": "string (encrypted)",
  "metadata": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### 3.2. Получение всех текстовых записей
```
GET /texts
```

**Headers:** Authorization required

**Query Parameters:**
- `limit` (optional)
- `offset` (optional)
- `search` (optional)

**Response:** `200 OK`
```json
{
  "items": [
    {
      "id": "uuid",
      "name": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ],
  "total": 50,
  "limit": 20,
  "offset": 0
}
```

---

#### 3.3. Получение конкретной текстовой записи
```
GET /texts/{id}
```

**Headers:** Authorization required

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "name": "string",
  "content": "string (encrypted)",
  "metadata": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### 3.4. Обновление текстовой записи
```
PUT /texts/{id}
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "name": "string",
  "content": "string",
  "metadata": "string"
}
```

**Response:** `200 OK`

---

#### 3.5. Удаление текстовой записи
```
DELETE /texts/{id}
```

**Headers:** Authorization required

**Response:** `204 No Content`

---

### 4. Управление бинарными данными

#### 4.1. Загрузка бинарных данных
```
POST /binaries
```

**Headers:** 
- Authorization required
- Content-Type: multipart/form-data

**Request Body:**
```
name: string
file: binary
metadata: string (optional)
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "name": "string",
  "filename": "string",
  "size": 1024,
  "content_type": "string",
  "metadata": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### 4.2. Получение списка бинарных данных
```
GET /binaries
```

**Headers:** Authorization required

**Query Parameters:**
- `limit` (optional)
- `offset` (optional)
- `search` (optional)

**Response:** `200 OK`
```json
{
  "items": [
    {
      "id": "uuid",
      "name": "string",
      "filename": "string",
      "size": 1024,
      "content_type": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ],
  "total": 30,
  "limit": 20,
  "offset": 0
}
```

---

#### 4.3. Скачивание бинарных данных
```
GET /binaries/{id}/download
```

**Headers:** Authorization required

**Response:** `200 OK`
- Content-Type: application/octet-stream
- Content-Disposition: attachment; filename="..."
- Body: binary data (encrypted)

---

#### 4.4. Получение метаданных бинарной записи
```
GET /binaries/{id}
```

**Headers:** Authorization required

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "name": "string",
  "filename": "string",
  "size": 1024,
  "content_type": "string",
  "metadata": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### 4.5. Обновление метаданных бинарной записи
```
PUT /binaries/{id}
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "name": "string",
  "metadata": "string"
}
```

**Response:** `200 OK`

---

#### 4.6. Удаление бинарных данных
```
DELETE /binaries/{id}
```

**Headers:** Authorization required

**Response:** `204 No Content`

---

### 5. Управление данными банковских карт

#### 5.1. Создание записи банковской карты
```
POST /cards
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "name": "string",
  "card_number": "string",
  "cardholder_name": "string",
  "expiry_date": "string",
  "cvv": "string",
  "bank_name": "string" (optional),
  "metadata": "string" (optional)
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "name": "string",
  "card_number": "string (encrypted, masked)",
  "cardholder_name": "string",
  "expiry_date": "string (encrypted)",
  "bank_name": "string",
  "metadata": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### 5.2. Получение всех записей банковских карт
```
GET /cards
```

**Headers:** Authorization required

**Query Parameters:**
- `limit` (optional)
- `offset` (optional)
- `search` (optional)

**Response:** `200 OK`
```json
{
  "items": [
    {
      "id": "uuid",
      "name": "string",
      "card_number": "string (masked: **** **** **** 1234)",
      "bank_name": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ],
  "total": 10,
  "limit": 20,
  "offset": 0
}
```

---

#### 5.3. Получение конкретной записи банковской карты
```
GET /cards/{id}
```

**Headers:** Authorization required

**Response:** `200 OK`
```json
{
  "id": "uuid",
  "name": "string",
  "card_number": "string (encrypted)",
  "cardholder_name": "string",
  "expiry_date": "string (encrypted)",
  "cvv": "string (encrypted)",
  "bank_name": "string",
  "metadata": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### 5.4. Обновление записи банковской карты
```
PUT /cards/{id}
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "name": "string",
  "card_number": "string",
  "cardholder_name": "string",
  "expiry_date": "string",
  "cvv": "string",
  "bank_name": "string",
  "metadata": "string"
}
```

**Response:** `200 OK`

---

#### 5.5. Удаление записи банковской карты
```
DELETE /cards/{id}
```

**Headers:** Authorization required

**Response:** `204 No Content`

---

### 6. Синхронизация данных

#### 6.1. Получение изменений с момента последней синхронизации
```
GET /sync
```

**Headers:** Authorization required

**Query Parameters:**
- `since` - timestamp последней синхронизации (optional)

**Response:** `200 OK`
```json
{
  "timestamp": "timestamp",
  "changes": {
    "credentials": [
      {
        "id": "uuid",
        "operation": "create|update|delete",
        "data": {...}
      }
    ],
    "texts": [...],
    "binaries": [...],
    "cards": [...]
  }
}
```

---

#### 6.2. Отправка локальных изменений на сервер
```
POST /sync
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "changes": {
    "credentials": [
      {
        "local_id": "uuid",
        "operation": "create|update|delete",
        "data": {...}
      }
    ],
    "texts": [...],
    "binaries": [...],
    "cards": [...]
  }
}
```

**Response:** `200 OK`
```json
{
  "timestamp": "timestamp",
  "conflicts": [
    {
      "type": "credentials|texts|binaries|cards",
      "local_id": "uuid",
      "server_id": "uuid",
      "conflict_type": "concurrent_update",
      "server_version": {...}
    }
  ],
  "synced_ids": {
    "local_id_1": "server_uuid_1",
    "local_id_2": "server_uuid_2"
  }
}
```

---

### 7. Пользовательский профиль

#### 7.1. Получение информации о пользователе
```
GET /user/profile
```

**Headers:** Authorization required

**Response:** `200 OK`
```json
{
  "user_id": "uuid",
  "username": "string",
  "email": "string",
  "created_at": "timestamp",
  "last_sync": "timestamp",
  "storage_used": 1024,
  "storage_limit": 10485760
}
```

---

#### 7.2. Обновление профиля пользователя
```
PUT /user/profile
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "email": "string"
}
```

**Response:** `200 OK`

---

#### 7.3. Изменение пароля
```
POST /user/change-password
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "old_password": "string",
  "new_password": "string"
}
```

**Response:** `200 OK`

**Errors:**
- `401` - Неверный старый пароль

---

#### 7.4. Удаление аккаунта
```
DELETE /user/account
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "password": "string",
  "confirmation": "DELETE_MY_ACCOUNT"
}
```

**Response:** `204 No Content`

---

### 8. Служебные эндпоинты

#### 8.1. Проверка здоровья сервера
```
GET /health
```

**Response:** `200 OK`
```json
{
  "status": "ok",
  "version": "1.0.0",
  "build_date": "2026-01-12T10:00:00Z"
}
```

---

#### 8.2. Получение версии API
```
GET /version
```

**Response:** `200 OK`
```json
{
  "api_version": "v1",
  "server_version": "1.0.0",
  "build_date": "2026-01-12T10:00:00Z",
  "commit": "abc123"
}
```

---

## Дополнительные эндпоинты (необязательные функции)

### 9. OTP (One Time Password)

#### 9.1. Создание OTP записи
```
POST /otp
```

**Headers:** Authorization required

**Request Body:**
```json
{
  "name": "string",
  "secret": "string",
  "issuer": "string" (optional),
  "account": "string" (optional),
  "algorithm": "SHA1|SHA256|SHA512",
  "digits": 6,
  "period": 30,
  "metadata": "string" (optional)
}
```

**Response:** `201 Created`

---

#### 9.2. Получение всех OTP записей
```
GET /otp
```

**Headers:** Authorization required

**Response:** `200 OK`

---

#### 9.3. Генерация OTP кода
```
GET /otp/{id}/generate
```

**Headers:** Authorization required

**Response:** `200 OK`
```json
{
  "code": "123456",
  "valid_until": "timestamp"
}
```

---

#### 9.4. Обновление OTP записи
```
PUT /otp/{id}
```

**Headers:** Authorization required

**Response:** `200 OK`

---

#### 9.5. Удаление OTP записи
```
DELETE /otp/{id}
```

**Headers:** Authorization required

**Response:** `204 No Content`

---

## Коды ответов

- `200 OK` - Успешный запрос
- `201 Created` - Ресурс создан
- `204 No Content` - Успешный запрос без тела ответа
- `400 Bad Request` - Невалидные данные запроса
- `401 Unauthorized` - Требуется аутентификация
- `403 Forbidden` - Доступ запрещен
- `404 Not Found` - Ресурс не найден
- `409 Conflict` - Конфликт данных
- `413 Payload Too Large` - Слишком большой размер данных
- `429 Too Many Requests` - Превышен лимит запросов
- `500 Internal Server Error` - Внутренняя ошибка сервера
- `503 Service Unavailable` - Сервис недоступен

---

## Безопасность

1. **Шифрование данных**: Все чувствительные данные (пароли, CVV, тексты, бинарные данные) шифруются на стороне клиента перед отправкой на сервер.

2. **HTTPS**: Все запросы должны выполняться через HTTPS.

3. **JWT токены**: Используются для аутентификации. Access token имеет короткий срок жизни (1 час), refresh token - более длительный.

4. **Rate Limiting**: Ограничение количества запросов для защиты от brute-force атак.

5. **Валидация входных данных**: Все входные данные валидируются на стороне сервера.

---

## Примечания

- Все timestamp'ы в формате ISO 8601: `2026-01-12T10:00:00Z`
- UUID используется для идентификаторов
- Пагинация по умолчанию: limit=20, offset=0
- Максимальный размер бинарных файлов должен быть ограничен (например, 10MB)
