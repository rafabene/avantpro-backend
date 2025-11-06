-- Migration: create_activation_tokens_table
-- Rollback: Drop activation_tokens table

DROP TABLE IF EXISTS activation_tokens CASCADE;
