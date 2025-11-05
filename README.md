# AvantPro Backend

Backend em Go seguindo Clean Architecture para gerenciamento de assinaturas.

## 📋 Pré-requisitos

- Go 1.25+
- PostgreSQL 16+
- Redis 7+
- Docker e Docker Compose (opcional)
- golang-migrate (para migrations)

## 🚀 Quick Start

### 1. Instalar dependências

```bash
make deps
make install-tools
```

### 2. Configurar ambiente

```bash
cp .env.example .env
# Edite .env com suas configurações
```

### 3. Iniciar banco de dados (Docker)

```bash
make docker/up
```

### 4. Rodar migrations

```bash
make db/migrate-up
```

### 5. Iniciar aplicação

```bash
# Desenvolvimento (hot reload)
make dev

# Ou build e run
make run
```

A API estará disponível em `http://localhost:8080`

## 📁 Estrutura do Projeto

```
avantpro-backend/
├── cmd/
│   └── api/              # Entry point
├── internal/
│   ├── domain/           # Domain Layer (entities, interfaces)
│   ├── services/         # Service Layer (business logic)
│   ├── handlers/         # Presentation Layer (HTTP handlers)
│   ├── infrastructure/   # Infrastructure Layer (DB, external services)
│   └── pkg/              # Shared packages
├── tests/                # Integration e E2E tests
├── configs/              # Configuration files
└── docs/                 # Documentation
```

## 🛠️ Comandos Principais

### Desenvolvimento

```bash
make dev              # Hot reload com Air
make run              # Build e executar
make build            # Build binário
```

### Testes

```bash
make test/all         # Todos os testes
make test/unit        # Unit tests
make test/integration # Integration tests
make test/e2e         # E2E tests
make test/coverage    # Coverage report
```

### Database

```bash
make db/migration name=create_users_table  # Criar migration
make db/migrate-up                         # Aplicar migrations
make db/migrate-down                       # Reverter última
make db/migrate-version                    # Ver versão atual
make db/reset                              # Reset completo
```

### Qualidade de Código

```bash
make lint             # Rodar linter
make lint-fix         # Fix automático
make fmt              # Formatar código
```

### Docker

```bash
make docker/up        # Iniciar serviços
make docker/down      # Parar serviços
make docker/logs      # Ver logs
```

## 🏗️ Arquitetura

Este projeto segue **Clean Architecture** com as seguintes camadas:

### Domain Layer
- Entidades de negócio puras
- Value Objects
- Interfaces (repositories, gateways, ports)
- Sem dependências externas

### Service Layer
- Casos de uso
- Lógica de negócio
- Orquestração de operações

### Presentation Layer
- HTTP handlers (Gin)
- DTOs e validação
- Middlewares

### Infrastructure Layer
- Implementações de repositórios (PostgreSQL/GORM)
- Gateways externos (Email, Payment)
- Configuração e logging

## 🔐 Autenticação

- JWT (Access + Refresh tokens)
- OAuth2/OIDC (Google, GitHub)
- RBAC (Role-Based Access Control)

## 📚 Documentação

**Requisitos Funcionais** (O QUE o sistema faz):
- [Autenticação e Autorização](specs/functional/auth.md)

**Especificações Técnicas** (COMO funciona):
- [Arquitetura](specs/technical/architecture.md)
- [Segurança (JWT/OAuth2)](specs/technical/security.md)
- [Database & Migrations](specs/technical/database.md)
- [Validação e i18n](specs/technical/validation-i18n.md)
- [Testes](specs/technical/testing.md)

**Guias de Desenvolvimento**:
- [Ferramentas](specs/development/tooling.md)

## 🤝 Contribuindo

1. Rode `make pre-commit` antes de commitar
2. Escreva testes para novas features
3. Mantenha coverage > 80%
4. Siga os padrões do projeto

