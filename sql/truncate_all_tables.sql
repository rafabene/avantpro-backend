-- Script para limpar todas as tabelas do banco de dados
-- Usado pelo comando 'make populate-test'

-- Limpar tabelas dependentes primeiro (ordem de dependência)
TRUNCATE TABLE notification_preferences RESTART IDENTITY CASCADE;
TRUNCATE TABLE notifications RESTART IDENTITY CASCADE;
TRUNCATE TABLE organization_invites RESTART IDENTITY CASCADE;
TRUNCATE TABLE organization_members RESTART IDENTITY CASCADE;
TRUNCATE TABLE organizations RESTART IDENTITY CASCADE;
TRUNCATE TABLE password_reset_tokens RESTART IDENTITY CASCADE;
TRUNCATE TABLE profiles RESTART IDENTITY CASCADE;
TRUNCATE TABLE users RESTART IDENTITY CASCADE;

-- Mostrar confirmação
SELECT 'All tables cleared successfully!' as status;