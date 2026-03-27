-- Seed data for local development and testing.

-- Test user (login: admin / password)
INSERT INTO users (username, password_hash)
VALUES ('admin', '$2a$10$placeholder_hash_for_dev')
ON CONFLICT (username) DO NOTHING;

-- Pre-registered example robot for AUTH flow testing.
-- Private key (hex): c55608b70c4a9f3b43bd1d23e86aaf4c3b2f4b823f54dc34ac668e85363ef2e1
-- Public key (hex):  b702036ee61847fdabecc07ce7da7b432c39aba98d1114c1c6f6f3f586ba98aa
INSERT INTO robots (uuid, public_key, device_type)
VALUES ('example-001', 'b702036ee61847fdabecc07ce7da7b432c39aba98d1114c1c6f6f3f586ba98aa', 'example_robot')
ON CONFLICT (uuid) DO NOTHING;
