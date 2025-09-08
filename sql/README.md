# Scripts SQL

Esta pasta contém scripts SQL utilizados para operações de banco de dados.

## Scripts Disponíveis

### `truncate_all_tables.sql`
- **Função**: Limpa todas as tabelas do banco de dados
- **Uso**: Executado pelo comando `make populate-test`
- **Características**:
  - Remove todos os dados de todas as tabelas
  - Reseta os IDs (RESTART IDENTITY)
  - Usa CASCADE para respeitar dependências
  - Executa na ordem correta das dependências

### `create_test_data.sql`
- **Função**: Script de verificação para dados de teste
- **Uso**: Verificação após limpeza das tabelas
- **Características**:
  - Verifica se o banco está vazio
  - Não cria usuários diretamente (senha precisa ser hasheada)
  - Usuários de teste são criados via API

## Como Usar

### Via Makefile (Recomendado)
```bash
# Limpar banco e criar usuário de teste
make populate-test
```

### Via Docker (Manual)
```bash
# Executar script de limpeza
docker exec -i avantpro-backend-postgres psql -U postgres -d avantpro_backend -f /path/to/truncate_all_tables.sql

# Verificar status
docker exec -i avantpro-backend-postgres psql -U postgres -d avantpro_backend -f /path/to/create_test_data.sql
```

### Via Cliente PostgreSQL
```bash
# Conectar ao banco
make db/shell

# Executar scripts
\i sql/truncate_all_tables.sql
\i sql/create_test_data.sql
```

## Estrutura das Tabelas

### Ordem de Dependência (para limpeza)
1. `notification_preferences` (depende de organizations)
2. `notifications` (depende de users e organizations) 
3. `organization_invites` (depende de organizations)
4. `organization_members` (depende de users e organizations)
5. `organizations` (depende de users como creator)
6. `password_reset_tokens` (depende de users)
7. `profiles` (depende de users)
8. `users` (tabela base)

## Notas de Segurança

- ⚠️ **ATENÇÃO**: Os scripts de limpeza removem TODOS os dados
- 🔒 **Produção**: Nunca execute scripts de limpeza em produção
- 🧪 **Desenvolvimento**: Use apenas em ambiente de desenvolvimento/teste
- 🔑 **Senhas**: Usuários de teste devem ser criados via API para hash correto