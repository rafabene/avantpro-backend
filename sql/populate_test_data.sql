-- Script completo para popular dados de teste
-- Usado pelo comando 'make populate-test'

-- 1. Criar usuário de teste
-- Senha: 123456 (hash: $2a$10$qgQQKHr8q3I8D7F.BcBT1e2Gp5crqdS8HaMqB5x1yCC/.QXR7U3KW)
INSERT INTO users (id, username, name, password, created_at, updated_at) 
VALUES (gen_random_uuid(), 'rafabene@gmail.com', 'Rafael Benevides', '$2a$10$qgQQKHr8q3I8D7F.BcBT1e2Gp5crqdS8HaMqB5x1yCC/.QXR7U3KW', NOW(), NOW());

-- 2. Criar organizações de teste e adicionar o usuário como admin
WITH user_info AS (
    SELECT id FROM users WHERE username = 'rafabene@gmail.com'
), 
org1 AS (
    INSERT INTO organizations (id, name, description, created_by, created_at, updated_at) 
    SELECT gen_random_uuid(), 'AvantPro Tecnologia', 'Empresa de desenvolvimento de software', id, NOW(), NOW() 
    FROM user_info 
    RETURNING id, created_by
), 
org2 AS (
    INSERT INTO organizations (id, name, description, created_by, created_at, updated_at) 
    SELECT gen_random_uuid(), 'Consultoria Rafael', 'Serviços de consultoria em TI', id, NOW(), NOW() 
    FROM user_info 
    RETURNING id, created_by
), 
member1 AS (
    INSERT INTO organization_members (id, organization_id, user_id, role, joined_at, created_at, updated_at) 
    SELECT gen_random_uuid(), org1.id, org1.created_by, 'admin', NOW(), NOW(), NOW() 
    FROM org1
), 
member2 AS (
    INSERT INTO organization_members (id, organization_id, user_id, role, joined_at, created_at, updated_at) 
    SELECT gen_random_uuid(), org2.id, org2.created_by, 'admin', NOW(), NOW(), NOW() 
    FROM org2
) 
SELECT 'Organizations and memberships created successfully' as result;