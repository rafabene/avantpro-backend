-- Script para verificar dados de teste criados
-- Usado pelo comando 'make populate-test' para verificar se os dados foram criados corretamente

-- Verificar dados criados
SELECT 'USERS' as table_name, COUNT(*) as total_records FROM users
UNION ALL
SELECT 'ORGANIZATIONS' as table_name, COUNT(*) as total_records FROM organizations
UNION ALL
SELECT 'ORGANIZATION MEMBERS' as table_name, COUNT(*) as total_records FROM organization_members;

-- Mostrar detalhes do usuário de teste
SELECT 
    'TEST USER DETAILS' as info,
    u.username as email,
    u.name,
    u.created_at
FROM users u 
WHERE u.username = 'rafabene@gmail.com';

-- Mostrar organizações do usuário de teste
SELECT 
    'USER ORGANIZATIONS' as info,
    o.name as organization_name,
    o.description,
    om.role,
    o.created_at
FROM users u
JOIN organization_members om ON u.id = om.user_id
JOIN organizations o ON om.organization_id = o.id
WHERE u.username = 'rafabene@gmail.com'
ORDER BY o.created_at;