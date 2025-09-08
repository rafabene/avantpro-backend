-- Script para mostrar o status de todas as tabelas
-- Mostra quantos registros existem em cada tabela

SELECT 
    'users' as table_name,
    COUNT(*) as record_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ Empty'
        ELSE '📊 Has data'
    END as status
FROM users

UNION ALL

SELECT 
    'profiles' as table_name,
    COUNT(*) as record_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ Empty'
        ELSE '📊 Has data'
    END as status
FROM profiles

UNION ALL

SELECT 
    'organizations' as table_name,
    COUNT(*) as record_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ Empty'
        ELSE '📊 Has data'
    END as status
FROM organizations

UNION ALL

SELECT 
    'organization_members' as table_name,
    COUNT(*) as record_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ Empty'
        ELSE '📊 Has data'
    END as status
FROM organization_members

UNION ALL

SELECT 
    'organization_invites' as table_name,
    COUNT(*) as record_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ Empty'
        ELSE '📊 Has data'
    END as status
FROM organization_invites

UNION ALL

SELECT 
    'notifications' as table_name,
    COUNT(*) as record_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ Empty'
        ELSE '📊 Has data'
    END as status
FROM notifications

UNION ALL

SELECT 
    'notification_preferences' as table_name,
    COUNT(*) as record_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ Empty'
        ELSE '📊 Has data'
    END as status
FROM notification_preferences

UNION ALL

SELECT 
    'password_reset_tokens' as table_name,
    COUNT(*) as record_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ Empty'
        ELSE '📊 Has data'
    END as status
FROM password_reset_tokens

ORDER BY table_name;