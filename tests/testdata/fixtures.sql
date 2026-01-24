-- Тестовые фикстуры для интеграционных тестов
-- Эти данные можно использовать для предзаполнения БД в тестах

-- Тестовые пользователи
-- Пароли: все используют "password123" (хеш должен быть сгенерирован в коде)
INSERT INTO users (id, username, email, hashed_password, created_at, updated_at)
VALUES 
    ('00000000-0000-0000-0000-000000000001', 'testuser1', 'user1@example.com', '$2a$10$YourHashedPasswordHere1', NOW(), NOW()),
    ('00000000-0000-0000-0000-000000000002', 'testuser2', 'user2@example.com', '$2a$10$YourHashedPasswordHere2', NOW(), NOW()),
    ('00000000-0000-0000-0000-000000000003', 'testuser3', 'user3@example.com', '$2a$10$YourHashedPasswordHere3', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Тестовые текстовые заметки
INSERT INTO texts (id, user_id, name, content, metadata, created_at, updated_at)
VALUES 
    ('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Test Note 1', 'Test content 1', '{"tags": ["test"]}', NOW(), NOW()),
    ('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Test Note 2', 'Test content 2', NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Тестовые credentials
INSERT INTO credentials (id, user_id, name, login, password, url, metadata, created_at, updated_at)
VALUES 
    ('20000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Gmail', 'user1@gmail.com', 'encrypted_password', 'https://gmail.com', NULL, NOW(), NOW()),
    ('20000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002', 'GitHub', 'user2', 'encrypted_password', 'https://github.com', NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Тестовые карты
INSERT INTO cards (id, user_id, name, card_number, cardholder_name, expiry_date, cvv, bank_name, metadata, created_at, updated_at)
VALUES 
    ('30000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Visa Card', '4111111111111111', 'John Doe', '12/25', '123', 'Test Bank', NULL, NOW(), NOW()),
    ('30000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002', 'MasterCard', '5555555555554444', 'Jane Smith', '01/26', '456', 'Another Bank', NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Тестовые бинарные файлы (metadata only, файлы на диске не создаются)
INSERT INTO binaries (id, user_id, name, filename, size, content_type, storage_path, checksum, metadata, created_at, updated_at)
VALUES 
    ('40000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Test PDF', 'document.pdf', 1024, 'application/pdf', '/storage/test/doc.pdf', 'abc123checksum', NULL, NOW(), NOW()),
    ('40000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002', 'Test Image', 'photo.jpg', 2048, 'image/jpeg', '/storage/test/photo.jpg', 'def456checksum', NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
