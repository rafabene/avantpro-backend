# AvantPro Backend

API para gerenciamento de usuários com suporte a perfil completo, desenvolvida em Go usando arquitetura de três camadas.

## 📋 Características

- **CRUD de Usuários**: Criar, listar, buscar por ID/username, atualizar e deletar usuários
- **Perfil de Usuário**: Endereço completo (rua, cidade, bairro, CEP) e telefone
- **Segurança**: Senhas criptografadas com bcrypt
- **Validação**: Validação completa de dados usando go-playground/validator
- **Paginação e Ordenação**: Lista paginada com ordenação por diferentes campos
- **Documentação**: Swagger/OpenAPI gerado automaticamente
- **Testes**: Cobertura completa com testes unitários, de repositório e integração

## 🛠 Tecnologias

- **Go 1.24**: Linguagem principal
- **Gin**: Framework web HTTP
- **GORM**: ORM para PostgreSQL
- **PostgreSQL**: Banco de dados principal
- **Swagger**: Documentação da API
- **Testcontainers**: Testes com PostgreSQL real
- **UUID**: Chaves primárias com PostgreSQL gen_random_uuid()

## 🚀 Endpoints da API

### Usuários

- `POST /api/v1/users` - Criar usuário
- `GET /api/v1/users` - Listar usuários (paginado, com ordenação)
- `GET /api/v1/users/{id}` - Buscar usuário por ID
- `GET /api/v1/users/username/{username}` - Buscar usuário por username (email)
- `PUT /api/v1/users/{id}` - Atualizar usuário
- `DELETE /api/v1/users/{id}` - Deletar usuário (soft delete)

### Parâmetros de Consulta (Lista)

- `page`: Número da página (padrão: 1)
- `limit`: Itens por página (padrão: 50, máx: 100)
- `sortBy`: Campo para ordenação (name, username, createdAt, updatedAt)
- `sortOrder`: Ordem (asc, desc, padrão: desc)

## 📁 Estrutura do Projeto

```
avantpro-backend/
├── cmd/server/          # Entry point da aplicação
├── internal/
│   ├── config/          # Configuração baseada em ambiente
│   ├── controllers/     # Controllers HTTP
│   ├── database/        # Conexão e migrações PostgreSQL
│   ├── errors/          # Tratamento de erros RFC 7807
│   ├── models/          # Domain models e DTOs
│   ├── repositories/    # Data access layer
│   └── services/        # Business logic
├── tests/integration/   # Testes de integração
├── docs/               # Documentação Swagger
├── bin/                # Binários compilados
└── Makefile           # Comandos de desenvolvimento
```

## 🔧 Configuração

### Variáveis de Ambiente (.env)

```bash
# Environment
ENV=development

# Server
PORT=8080
TRUSTED_PROXIES=

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=avantpro_backend
DB_SSLMODE=disable
```

### Segurança em Produção

- **Swagger UI**: Disponível apenas em `ENV=development`
- **Senhas**: Sempre criptografadas com bcrypt
- **Trusted Proxies**: Configuráveis via `TRUSTED_PROXIES` para produção
- **SSL**: Recomendado `DB_SSLMODE=require` em produção

## 🏃‍♂️ Como Executar

### 1. Pré-requisitos

- Go 1.24+
- PostgreSQL (ou Docker)
- Make (opcional, mas recomendado)

### 2. Configuração do Ambiente

```bash
# Clone o projeto
git clone <repository-url>
cd avantpro-backend

# Instalar ferramentas de desenvolvimento
make install-tools

# Configurar banco de dados com Docker
make db/setup

# Configurar variáveis de ambiente
cp .env.example .env
# Edite .env conforme necessário
```

### 3. Executar Aplicação

```bash
# Desenvolvimento (com hot reload)
make dev

# Ou executar diretamente
make run

# Build e executar
make build
./bin/avantpro-backend
```

### 4. Acessar Documentação

- API: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html (apenas em desenvolvimento)
- Health Check: http://localhost:8080/health

## 🧪 Testes

```bash
# Executar todos os testes
make test

# Testes com cobertura
make test-coverage

# Verificação completa (fmt, vet, lint, test)
make check
```

## 📊 Exemplos de Uso

### Criar Usuário

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user@example.com",
    "name": "João Silva",
    "password": "password123",
    "profile": {
      "street": "Rua das Flores, 123",
      "city": "São Paulo",
      "district": "Centro",
      "zip_code": "01234567",
      "phone": "11987654321"
    }
  }'
```

### Buscar por Username

```bash
curl http://localhost:8080/api/v1/users/username/user@example.com
```

### Listar com Paginação e Ordenação

```bash
curl "http://localhost:8080/api/v1/users?page=1&limit=10&sortBy=name&sortOrder=asc"
```

## 🔍 Características Técnicas

### Arquitetura de Três Camadas

- **Controllers**: Manipulação de requisições HTTP e validação de entrada
- **Services**: Lógica de negócio e validações
- **Repositories**: Acesso a dados e operações de banco

### Segurança

- Senhas criptografadas com bcrypt
- Prevenção de SQL injection com whitelist de campos
- Configuração segura de trusted proxies
- Validação completa de entrada

### Validações

- Username deve ser email válido
- Nome: 2-100 caracteres
- Senha: mínimo 6 caracteres
- CEP: exatamente 8 dígitos
- Telefone: 10-15 caracteres

### Error Handling

Implementa RFC 7807 Problem Details para respostas de erro padronizadas:

```json
{
  "type": "https://avantpro-backend.com/errors/validation",
  "title": "Validation Error",
  "status": 400,
  "detail": "username is required",
  "instance": "/api/v1/users"
}
```

## 🛠 Comandos Disponíveis

```bash
# Desenvolvimento
make dev          # Servidor com hot reload
make run          # Executar aplicação
make build        # Compilar aplicação

# Testes
make test         # Executar testes
make test-coverage # Testes com cobertura

# Qualidade de Código
make fmt          # Formatar código
make lint         # Linting
make vet          # Análise estática
make check        # Verificação completa

# Banco de Dados
make db/setup     # Iniciar PostgreSQL (Docker)
make db/teardown  # Parar PostgreSQL
make db/shell     # Conectar ao banco

# Documentação
make swagger      # Gerar documentação Swagger

# Utilitários
make clean        # Limpar artefatos
make help         # Mostrar ajuda
```

## 📝 Licença

MIT License