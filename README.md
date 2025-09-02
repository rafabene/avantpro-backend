# AvantPro Backend

API para gerenciamento de usuários com suporte a perfil completo, desenvolvida em Go usando arquitetura de três camadas.

## 📋 Características

- **Autenticação JWT**: Login, registro e recuperação de senha com tokens JWT
- **Sistema de Organizações**: Criação, gerenciamento de membros e sistema de convites
- **Perfil de Usuário**: Endereço completo (rua, cidade, bairro, CEP) e telefone
- **Segurança**: Senhas criptografadas com bcrypt e autenticação baseada em tokens
- **Validação**: Validação completa de dados usando go-playground/validator
- **Paginação e Ordenação**: Lista paginada com ordenação por diferentes campos
- **CORS**: Configuração de CORS para integração com frontend
- **Documentação**: Swagger/OpenAPI gerado automaticamente
- **Testes**: Cobertura completa com testes unitários, de repositório e integração

## 🛠 Tecnologias

- **Go 1.24**: Linguagem principal
- **Gin**: Framework web HTTP com middleware CORS
- **GORM**: ORM para PostgreSQL
- **PostgreSQL**: Banco de dados principal
- **JWT**: Autenticação baseada em tokens (golang-jwt/jwt)
- **bcrypt**: Criptografia de senhas
- **Swagger**: Documentação da API
- **Testcontainers**: Testes com PostgreSQL real
- **UUID**: Chaves primárias com PostgreSQL gen_random_uuid()

## 🚀 Endpoints da API

### Autenticação

- `POST /api/v1/auth/login` - Fazer login e obter token JWT
- `POST /api/v1/auth/register` - Registrar novo usuário e obter token
- `POST /api/v1/auth/password-reset` - Solicitar reset de senha
- `POST /api/v1/auth/password-reset/confirm` - Confirmar reset de senha com token

### Organizações

- `POST /api/v1/organizations` - Criar organização
- `GET /api/v1/organizations/my` - Listar organizações criadas pelo usuário
- `GET /api/v1/organizations/memberships` - Listar organizações onde é membro
- `GET /api/v1/organizations/{id}` - Buscar organização por ID
- `PUT /api/v1/organizations/{id}` - Atualizar organização
- `DELETE /api/v1/organizations/{id}` - Deletar organização
- `GET /api/v1/organizations/{id}/members` - Listar membros da organização
- `PUT /api/v1/organizations/{id}/members/{userId}` - Atualizar role do membro
- `DELETE /api/v1/organizations/{id}/members/{userId}` - Remover membro
- `POST /api/v1/organizations/{id}/invites` - Convidar usuário
- `GET /api/v1/organizations/{id}/invites` - Listar convites da organização
- `POST /api/v1/organizations/invites/token/{token}/accept` - Aceitar convite
- `DELETE /api/v1/organizations/invites/id/{inviteId}` - Revogar convite
- `POST /api/v1/organizations/invites/id/{inviteId}/resend` - Reenviar convite

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
│   ├── controllers/     # Controllers HTTP (auth, organization)
│   ├── database/        # Conexão e migrações PostgreSQL
│   ├── errors/          # Tratamento de erros RFC 7807
│   ├── models/          # Domain models e DTOs
│   ├── repositories/    # Data access layer
│   └── services/        # Business logic (auth, organization)
├── tests/integration/   # Testes de integração
├── docs/               # Documentação Swagger gerada
├── bin/                # Binários compilados
├── tmp/                # Arquivos temporários (air)
├── CLAUDE.md           # Instruções para Claude Code
├── Makefile           # Comandos de desenvolvimento
└── go.mod             # Dependências Go
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

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_EXPIRES_IN=24h
```

### Segurança em Produção

- **Swagger UI**: Disponível apenas em `ENV=development`
- **JWT Secret**: OBRIGATÓRIO alterar `JWT_SECRET` em produção
- **Senhas**: Sempre criptografadas com bcrypt
- **CORS**: Configurado para `http://localhost:4200` (ajustar para produção)
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
# Crie um arquivo .env na raiz do projeto com as variáveis necessárias
# (veja a seção "Variáveis de Ambiente" abaixo)
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

- **API**: http://localhost:8080
- **Swagger UI**: http://localhost:8080/swagger/index.html (apenas em desenvolvimento)
- **Health Check**: http://localhost:8080/health
- **Documentação da API**: Todas as rotas documentadas com Swagger/OpenAPI

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

### Autenticação

#### Registrar Usuário
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
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

#### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user@example.com",
    "password": "password123"
  }'
```

#### Solicitar Reset de Senha
```bash
curl -X POST http://localhost:8080/api/v1/auth/password-reset \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

### Organizações

#### Criar Organização
```bash
curl -X POST http://localhost:8080/api/v1/organizations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "Minha Organização",
    "description": "Descrição da organização"
  }'
```

#### Listar Organizações do Usuário
```bash
curl "http://localhost:8080/api/v1/organizations/my?page=1&limit=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Convidar Usuário para Organização
```bash
curl -X POST http://localhost:8080/api/v1/organizations/{id}/invites \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "email": "user@example.com",
    "role": "user"
  }'
```

## 🔍 Características Técnicas

### Arquitetura de Três Camadas

- **Controllers**: Manipulação de requisições HTTP e validação de entrada
- **Services**: Lógica de negócio e validações
- **Repositories**: Acesso a dados e operações de banco

### Segurança

- **Autenticação JWT**: Tokens seguros para autenticação de usuários
- **Senhas criptografadas**: bcrypt com salt automático
- **Prevenção SQL Injection**: Whitelist de campos para ordenação
- **CORS configurado**: Controle de origem para requests cross-domain
- **Trusted Proxies**: Configuração segura para ambientes de produção
- **Validação completa**: Entrada sanitizada em todos os endpoints

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
  "instance": "/api/v1/organizations"
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