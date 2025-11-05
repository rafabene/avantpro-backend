# Migrations e Database

**Versão**: 2.0
**Data**: 05/11/2025

---

## 1. Visão Geral

Estratégia de migrations e gerenciamento de banco de dados:
- **golang-migrate/migrate** para migrations versionadas
- **SQL puro** para controle total e clareza
- **GORM** para queries e operações de dados
- **Separação** entre models GORM e entities de domínio

---

## 2. Setup de Migrations

### 2.1 Instalação

```bash
# CLI do golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Verificar instalação
migrate -version
```

### 2.2 Estrutura de Diretórios

```
internal/infrastructure/persistence/migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_subscriptions_table.up.sql
├── 000002_create_subscriptions_table.down.sql
├── 000003_add_users_email_index.up.sql
├── 000003_add_users_email_index.down.sql
├── 000004_create_payments_table.up.sql
├── 000004_create_payments_table.down.sql
└── ...
```

**Convenções:**
- Numeração sequencial com 6 dígitos: `000001`, `000002`, etc
- Sufixo `.up.sql` para aplicar migration
- Sufixo `.down.sql` para reverter migration
- Nome descritivo da mudança: `create_users_table`, `add_email_index`

---

## 3. Criando Migrations

### 3.1 Comandos

```bash
# Criar nova migration (cria arquivos .up.sql e .down.sql)
migrate create -ext sql -dir internal/infrastructure/persistence/migrations -seq create_users_table

# Aplicar todas as migrations pendentes
migrate -path internal/infrastructure/persistence/migrations -database "postgresql://user:pass@localhost:5432/dbname?sslmode=disable" up

# Aplicar N migrations
migrate -path internal/infrastructure/persistence/migrations -database "..." up 2

# Reverter última migration
migrate -path internal/infrastructure/persistence/migrations -database "..." down 1

# Reverter todas
migrate -path internal/infrastructure/persistence/migrations -database "..." down

# Ver versão atual
migrate -path internal/infrastructure/persistence/migrations -database "..." version

# Forçar versão (use com cautela!)
migrate -path internal/infrastructure/persistence/migrations -database "..." force 5
```

### 3.2 Makefile para Migrations

```makefile
# Makefile

# Database configs
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASS ?= postgres
DB_NAME ?= avantpro
DB_URL := postgresql://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

MIGRATIONS_DIR := internal/infrastructure/persistence/migrations

.PHONY: db/migration db/migrate-up db/migrate-down db/migrate-version db/migrate-force

# Criar nova migration
# Uso: make db/migration name=create_users_table
db/migration:
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make db/migration name=create_users_table"; \
		exit 1; \
	fi
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)
	@echo "Migration created: $(MIGRATIONS_DIR)/000XXX_$(name).{up,down}.sql"

# Aplicar migrations
db/migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

# Reverter última migration
db/migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

# Ver versão atual
db/migrate-version:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

# Forçar versão específica (cuidado!)
# Uso: make db/migrate-force version=5
db/migrate-force:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required. Usage: make db/migrate-force version=5"; \
		exit 1; \
	fi
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(version)

# Aplicar migrations em ambiente de produção
migrate-up-prod:
	@echo "Applying migrations to PRODUCTION..."
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL_PROD)" up; \
	fi
```

---

## 4. Exemplos de Migrations

### 4.1 Create Table

```sql
-- 000001_create_users_table.up.sql

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from now())
);

-- Índices
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_created_at ON users(created_at);

-- Comentários
COMMENT ON TABLE users IS 'User accounts';
COMMENT ON COLUMN users.email IS 'User email (unique)';
COMMENT ON COLUMN users.role IS 'User role: admin, user, guest';
```

```sql
-- 000001_create_users_table.down.sql

DROP TABLE IF EXISTS users CASCADE;
```

### 4.2 Alter Table - Add Column

```sql
-- 000005_add_users_avatar_url.up.sql

ALTER TABLE users
ADD COLUMN avatar_url VARCHAR(500);

COMMENT ON COLUMN users.avatar_url IS 'URL of user avatar image';
```

```sql
-- 000005_add_users_avatar_url.down.sql

ALTER TABLE users
DROP COLUMN IF EXISTS avatar_url;
```

### 4.3 Alter Table - Modify Column

```sql
-- 000006_increase_users_name_length.up.sql

ALTER TABLE users
ALTER COLUMN name TYPE VARCHAR(500);
```

```sql
-- 000006_increase_users_name_length.down.sql

-- CUIDADO: pode causar perda de dados se nomes > 255 chars
ALTER TABLE users
ALTER COLUMN name TYPE VARCHAR(255);
```

### 4.4 Create Index

```sql
-- 000007_add_users_name_search_index.up.sql

-- Índice para busca case-insensitive
CREATE INDEX idx_users_name_lower ON users(LOWER(name));

-- Índice composto
CREATE INDEX idx_users_role_created_at ON users(role, created_at DESC);
```

```sql
-- 000007_add_users_name_search_index.down.sql

DROP INDEX IF EXISTS idx_users_name_lower;
DROP INDEX IF EXISTS idx_users_role_created_at;
```

### 4.5 Foreign Keys e Relacionamentos

```sql
-- 000002_create_subscriptions_table.up.sql

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    plan VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    started_at BIGINT NOT NULL,
    ends_at BIGINT,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from now()),

    -- Foreign key com ON DELETE CASCADE
    CONSTRAINT fk_subscriptions_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

-- Índices
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_ends_at ON subscriptions(ends_at);

COMMENT ON TABLE subscriptions IS 'User subscriptions';
```

```sql
-- 000002_create_subscriptions_table.down.sql

DROP TABLE IF EXISTS subscriptions CASCADE;
```

### 4.6 Data Migration

```sql
-- 000010_migrate_old_roles.up.sql

-- Migrar dados antigos
UPDATE users
SET role = 'user'
WHERE role IS NULL OR role = '';

-- Adicionar constraint NOT NULL após garantir dados válidos
ALTER TABLE users
ALTER COLUMN role SET NOT NULL;
```

```sql
-- 000010_migrate_old_roles.down.sql

ALTER TABLE users
ALTER COLUMN role DROP NOT NULL;
```

### 4.7 Triggers e Functions

```sql
-- 000015_add_updated_at_trigger.up.sql

-- Function para atualizar updated_at automaticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = extract(epoch from now());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger para users
CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger para subscriptions
CREATE TRIGGER update_subscriptions_updated_at
BEFORE UPDATE ON subscriptions
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
```

```sql
-- 000015_add_updated_at_trigger.down.sql

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;
DROP FUNCTION IF EXISTS update_updated_at_column();
```

---

## 5. GORM Models vs Domain Entities

### 5.1 Separação de Responsabilidades

**Domain Entity** (regras de negócio, lógica pura):
```go
// internal/domain/entities/user.go
package entities

import (
    "time"
    "avantpro-backend/internal/domain/valueobjects"
)

type User struct {
    ID           string
    Email        valueobjects.Email  // Value Object
    Name         string
    PasswordHash string
    Role         Role
    AvatarURL    *string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// Métodos de domínio
func (u *User) IsAdmin() bool {
    return u.Role == RoleAdmin
}

func (u *User) CanAccessResource(resourceID string) bool {
    // Lógica de negócio
    return true
}
```

**GORM Model** (mapeamento de banco, persistência):
```go
// internal/infrastructure/persistence/postgres/models.go
package postgres

import "time"

type UserModel struct {
    ID           string  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email        string  `gorm:"type:varchar(255);uniqueIndex;not null"`
    Name         string  `gorm:"type:varchar(500);not null"`
    PasswordHash string  `gorm:"type:varchar(255);not null"`
    Role         string  `gorm:"type:varchar(50);not null;index"`
    AvatarURL    *string `gorm:"type:varchar(500)"`
    CreatedAt    int64   `gorm:"autoCreateTime;index"`
    UpdatedAt    int64   `gorm:"autoUpdateTime"`
}

func (UserModel) TableName() string {
    return "users"
}
```

### 5.2 Conversores (Mappers)

```go
// internal/infrastructure/persistence/postgres/user_repository.go
package postgres

import (
    "avantpro-backend/internal/domain/entities"
    "avantpro-backend/internal/domain/valueobjects"
    "time"
)

// toModel converte entity para GORM model
func (r *UserRepository) toModel(user *entities.User) *UserModel {
    return &UserModel{
        ID:           user.ID,
        Email:        user.Email.String(),
        Name:         user.Name,
        PasswordHash: user.PasswordHash,
        Role:         string(user.Role),
        AvatarURL:    user.AvatarURL,
        CreatedAt:    user.CreatedAt.Unix(),
        UpdatedAt:    user.UpdatedAt.Unix(),
    }
}

// toEntity converte GORM model para entity
func (r *UserRepository) toEntity(model *UserModel) (*entities.User, error) {
    email, err := valueobjects.NewEmail(model.Email)
    if err != nil {
        return nil, err
    }

    return &entities.User{
        ID:           model.ID,
        Email:        email,
        Name:         model.Name,
        PasswordHash: model.PasswordHash,
        Role:         entities.Role(model.Role),
        AvatarURL:    model.AvatarURL,
        CreatedAt:    time.Unix(model.CreatedAt, 0),
        UpdatedAt:    time.Unix(model.UpdatedAt, 0),
    }, nil
}

// toEntities converte slice de models para entities
func (r *UserRepository) toEntities(models []*UserModel) ([]*entities.User, error) {
    entities := make([]*entities.User, 0, len(models))

    for _, model := range models {
        entity, err := r.toEntity(model)
        if err != nil {
            return nil, err
        }
        entities = append(entities, entity)
    }

    return entities, nil
}
```

---

## 6. Conexão com Database

### 6.1 Database Configuration

```go
// internal/infrastructure/config/database.go
package config

type DatabaseConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    DBName   string
    SSLMode  string
    MaxConns int
    MinConns int
    MaxIdleTime int
}

func (c *DatabaseConfig) DSN() string {
    return fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
    )
}
```

### 6.2 Database Connection

```go
// internal/infrastructure/persistence/postgres/connection.go
package postgres

import (
    "fmt"
    "log"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    "avantpro-backend/internal/infrastructure/config"
)

func NewDatabaseConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
    // GORM config
    gormConfig := &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
        NowFunc: func() time.Time {
            return time.Now().UTC()
        },
        // Desabilitar prepared statements para melhor performance em alguns casos
        PrepareStmt: false,
    }

    // Conectar
    db, err := gorm.Open(postgres.Open(cfg.DSN()), gormConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Configurar connection pool
    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get sql.DB: %w", err)
    }

    sqlDB.SetMaxOpenConns(cfg.MaxConns)
    sqlDB.SetMaxIdleConns(cfg.MinConns)
    sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxIdleTime) * time.Second)

    // Ping para verificar conexão
    if err := sqlDB.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    log.Println("Database connected successfully")
    return db, nil
}
```

---

## 7. Boas Práticas

### 7.1 Schema Design

**✅ DO:**
- UUIDs para primary keys (evita sequence locks)
- Timestamps em Unix (int64) para simplicidade
- Índices em foreign keys
- Índices em campos de busca frequente
- NOT NULL quando apropriado
- Default values sensatos
- Comentários em tabelas e colunas complexas

**❌ DON'T:**
- Auto-increment IDs em sistemas distribuídos
- ENUMs (preferir VARCHAR com check constraints)
- Colunas JSON para dados estruturados críticos
- Muitos índices desnecessários (impacto em writes)
- Soft deletes sem necessidade (adiciona complexidade)

### 7.2 Índices

```sql
-- Bons exemplos de índices

-- 1. Unique index para campos únicos
CREATE UNIQUE INDEX idx_users_email ON users(email);

-- 2. Index composto (ordem importa!)
-- Bom para queries: WHERE role = 'admin' ORDER BY created_at DESC
CREATE INDEX idx_users_role_created_at ON users(role, created_at DESC);

-- 3. Partial index (mais eficiente)
CREATE INDEX idx_active_subscriptions
ON subscriptions(user_id, ends_at)
WHERE status = 'active';

-- 4. Index para busca case-insensitive
CREATE INDEX idx_users_email_lower ON users(LOWER(email));

-- 5. GIN index para full-text search (PostgreSQL)
CREATE INDEX idx_users_name_fts ON users USING GIN(to_tsvector('english', name));
```

### 7.3 Migrations Checklist

- [ ] **Sempre criar arquivo .down.sql** (rollback)
- [ ] **Testar em ambiente de dev primeiro**
- [ ] **Testar rollback (.down.sql)**
- [ ] **Backupear dados antes de migrations destrutivas**
- [ ] **Usar transações quando possível**
- [ ] **Adicionar comentários para context**
- [ ] **Migrations idempotentes** (IF EXISTS, IF NOT EXISTS)
- [ ] **Evitar locks longos** (ALTER TABLE em tabelas grandes)
- [ ] **Notificar equipe antes de rodar em prod**

### 7.4 Performance Tips

**Connection Pool:**
```go
// Configurações recomendadas
sqlDB.SetMaxOpenConns(25)  // Máximo de conexões abertas
sqlDB.SetMaxIdleConns(5)   // Conexões idle no pool
sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Tempo máximo de vida
```

**Prepared Statements:**
```go
// Habilitar para queries repetidas
gormConfig := &gorm.Config{
    PrepareStmt: true,  // Reusar prepared statements
}
```

**Batch Operations:**
```go
// Usar CreateInBatches para múltiplos inserts
users := []*UserModel{user1, user2, user3, ...}
db.CreateInBatches(users, 100)  // Batches de 100
```

---

## 8. Migration CI/CD

### 8.1 GitHub Actions Example

```yaml
# .github/workflows/migrations.yml
name: Database Migrations

on:
  push:
    branches: [main]
    paths:
      - 'internal/infrastructure/persistence/migrations/**'

jobs:
  validate-migrations:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3

      - name: Install golang-migrate
        run: |
          curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
          sudo mv migrate /usr/local/bin/

      - name: Run migrations
        run: |
          migrate -path internal/infrastructure/persistence/migrations \
                  -database "postgresql://test:test@localhost:5432/testdb?sslmode=disable" \
                  up

      - name: Test rollback
        run: |
          migrate -path internal/infrastructure/persistence/migrations \
                  -database "postgresql://test:test@localhost:5432/testdb?sslmode=disable" \
                  down 1
```

---

## 9. Troubleshooting

### 9.1 Problemas Comuns

**Erro: "Dirty database version"**
```bash
# Acontece quando migration falha no meio

# Verificar versão atual
make db/migrate-version

# Forçar versão (se tiver certeza que pode)
make db/migrate-force version=5

# Depois aplicar novamente
make db/migrate-up
```

**Migration travada (lock)**
```sql
-- Ver locks ativos
SELECT * FROM pg_locks WHERE NOT granted;

-- Matar processo travado (cuidado!)
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'active' AND query LIKE '%migrate%';
```

**Rollback não funcionou**
```bash
# Verificar se arquivo .down.sql está correto
cat internal/infrastructure/persistence/migrations/000XXX_name.down.sql

# Executar SQL manualmente se necessário
psql -U user -d dbname -f internal/infrastructure/persistence/migrations/000XXX_name.down.sql
```

---

**Versão**: 2.0
**Última Atualização**: 05/11/2025
