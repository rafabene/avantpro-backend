# Multi-Tenancy - Requisitos Funcionais

**Versão**: 3.4
**Data**: 05/11/2025
**Changelog**:
- v3.4: Removido slug de organizations (identificação apenas por UUID + name)
- v3.3: Renomeado Tenant→Organization, adicionado UserAccount (1:1 com User), removidas seções de signup/invites/planos (delegadas para outras specs)
- v3.2: Removida Fase 6 (Observabilidade) - não é escopo de multi-tenancy
- v3.1: Removido RLS (Row-Level Security) - filtro explícito no código é preferível
- v3.0: Adicionado fluxo de login em 2 etapas (temp_token + select-tenant)

---

## 1. Visão Geral

**AvantPro é um sistema SaaS multi-tenant** estilo Netflix/Spotify onde todos os clientes usam o mesmo domínio e compartilham a mesma infraestrutura, operando de forma isolada através de dados segregados.

### 1.1 Modelo de Multi-Tenancy

**Tipo**: **Shared Database + Shared Schema** (Row-Level Isolation)
- **Um único domínio**: `app.avantpro.com.br`
- **Um único banco de dados**: PostgreSQL compartilhado
- **Um único schema**: Tabelas de negócio têm coluna `organization_id`
- **Zero configuração de infraestrutura**: Cliente se registra e começa a usar
- **Escalabilidade horizontal**: Adiciona servidores, não databases

### 1.2 Conceitos

**Organization (Empresa/Organização)**
- Representa uma empresa/cliente usando o sistema
- Exemplo: "Empresa ABC", "Startup XYZ"
- Cada organization tem: assinaturas, configurações isoladas
- Identificado por UUID único
- **É a raiz do isolamento de dados**

**Usuário (Global)**
- Entidade **GLOBAL**, não pertence a nenhuma organization
- Identificado por email único no sistema inteiro
- **Pode pertencer a múltiplas organizations simultaneamente**
- Autenticação: email + senha (única para todas as organizations)

**UserAccount (Dados da Conta - 1:1 com User)**
- Relacionamento 1:1 com User
- Armazena dados pessoais: nome completo, avatar, telefone, preferências
- Permite User ser minimalista (apenas email + senha para autenticação)
- Cada usuário tem exatamente um UserAccount

**OrganizationMember (Associação Usuário-Organization)**
- Relacionamento N:N entre User e Organization
- Define a **role específica** do usuário naquela organization
- Exemplo: João pode ser:
  - Admin na Empresa ABC (organization A)
  - User na Startup XYZ (organization B)
  - Guest na Consultoria (organization C)

**Isolamento por Dados**
- Todas as tabelas de negócio têm coluna `organization_id UUID NOT NULL`
- Repositories SEMPRE filtram queries com `WHERE organization_id = ?` (explícito no código)
- Middleware valida organization_id do JWT antes de cada request
- Testes automatizados garantem isolamento entre organizations

---

## 2. Identificação de Organization

### 2.1 Método de Identificação

**Domínio Único + Autenticação + Seleção de Organization**
```
Todos acessam: https://app.avantpro.com.br

Login do usuário NÃO determina automaticamente a organization:
- joao@email.com → pode pertencer a múltiplas organizations
- Usuário escolhe qual organization quer acessar após login
```

**Fluxo de Login (2 etapas)**:

**Etapa 1: Autenticação (POST /auth/login)**
```
1. Usuário envia email + senha para `POST /auth/login`
2. Sistema valida credenciais
3. Sistema busca todas as organizations do usuário:
   SELECT o.*, om.role
   FROM organizations o
   JOIN organization_members om ON om.organization_id = o.id
   WHERE om.user_id = ?

4. Caso A - Usuário tem 1 organization apenas:
   → Retorna JWT final (com organization_id) + refresh_token
   → Usuário vai direto para dashboard

5. Caso B - Usuário tem múltiplas organizations:
   → Retorna token temporário + lista de organizations
   → Frontend mostra tela: "Selecione a organização"
```

**Etapa 2: Seleção de Organization (POST /auth/select-organization)** - apenas se múltiplas organizations
```
1. Frontend envia: { organization_id } + temp_token no header
2. Sistema valida temp_token
3. Sistema valida que user é membro da organization escolhida
4. Sistema gera JWT final (com organization_id) + refresh_token
5. Usuário acessa dashboard da organization escolhida
```

**Todas as requests subsequentes usam `organization_id` do JWT**

### 2.2 Estrutura dos Tokens

**Token Temporário** (apenas para seleção de organization):
```json
{
  "sub": "user-uuid-123",
  "email": "joao@email.com",
  "type": "organization_selection",
  "exp": 1699123456  // 15 minutos
}
```

**JWT Final** (access_token - após selecionar organization):
```json
{
  "sub": "user-uuid-123",
  "email": "joao@email.com",
  "organization_id": "organization-uuid-abc",
  "organization_name": "Empresa ABC",
  "role": "admin",
  "permissions": ["users.read", "users.write"],
  "type": "access",
  "iat": 1699123456,
  "exp": 1699124356  // 15 minutos
}
```

**Refresh Token** (armazenado no Redis):
```json
{
  "sub": "user-uuid-123",
  "organization_id": "organization-uuid-abc",
  "type": "refresh",
  "exp": 1699728256  // 7 dias
}
```

**Middleware extrai dados do JWT**:
```go
ctx = context.WithValue(ctx, "user_id", claims.Sub)
ctx = context.WithValue(ctx, "organization_id", claims.OrganizationID)
ctx = context.WithValue(ctx, "role", claims.Role)  // role específica nesta organization
```

### 2.3 Endpoints de Autenticação

**POST /auth/login** - Autenticação inicial

**Request**:
```json
{
  "email": "joao@email.com",
  "password": "senha123"
}
```

**Response A** (1 organization apenas):
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "organization": {
    "id": "uuid-abc",
    "name": "Empresa ABC",
    "role": "admin"
  }
}
```

**Response B** (múltiplas organizations):
```json
{
  "requires_organization_selection": true,
  "temp_token": "eyJhbGc...",
  "organizations": [
    {
      "id": "uuid-abc",
      "name": "Empresa ABC",
      "role": "admin"
    },
    {
      "id": "uuid-xyz",
      "name": "Startup XYZ",
      "role": "user"
    }
  ]
}
```

---

**POST /auth/select-organization** - Seleção de organization (requer temp_token)

**Request**:
```http
POST /auth/select-organization
Authorization: Bearer <temp_token>
Content-Type: application/json

{
  "organization_id": "uuid-abc"
}
```

**Response**:
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "organization": {
    "id": "uuid-abc",
    "name": "Empresa ABC",
    "role": "admin"
  }
}
```

---

**POST /auth/switch-organization** - Trocar de organização (requer access_token)

**Request**:
```http
POST /auth/switch-organization
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "organization_id": "uuid-xyz"
}
```

**Response**:
```json
{
  "access_token": "eyJhbGc...",  // novo JWT com organization_id diferente
  "refresh_token": "...",
  "organization": {
    "id": "uuid-xyz",
    "name": "Startup XYZ",
    "role": "user"
  }
}
```

### 2.4 Lógica de Negócio do Fluxo de Login

**Cenário A: Usuário com 1 Organization**

```gherkin
Given um usuário cadastrado com email "user@example.com"
And o usuário pertence a exatamente 1 organization "Empresa ABC"
When ele envia POST /auth/login com credenciais válidas
Then o sistema valida email e senha
And busca organizations do usuário
And encontra 1 organization
And gera JWT final contendo:
  - user_id
  - organization_id da única organization
  - role do usuário naquela organization
And retorna access_token e refresh_token
And retorna dados da organization
And o usuário é redirecionado para dashboard
```

**Cenário B: Usuário com múltiplas Organizations**

```gherkin
Given um usuário cadastrado com email "user@example.com"
And o usuário pertence a 3 organizations:
  - "Empresa ABC" (role: admin)
  - "Startup XYZ" (role: user)
  - "Consultoria" (role: guest)
When ele envia POST /auth/login com credenciais válidas
Then o sistema valida email e senha
And busca organizations do usuário
And encontra 3 organizations
And gera token temporário (tipo: "organization_selection") contendo apenas user_id
And retorna lista de organizations com roles
And o frontend mostra modal de seleção
When o usuário seleciona "Empresa ABC"
And envia POST /auth/select-organization com organization_id
Then o sistema valida que usuário é membro da organization escolhida
And gera JWT final com organization_id = "Empresa ABC"
And retorna access_token e refresh_token
And o usuário acessa dashboard da "Empresa ABC"
```

**Cenário C: Trocar de Organization (Switch)**

```gherkin
Given um usuário autenticado atualmente na "Empresa ABC"
And o usuário também pertence à "Startup XYZ"
When ele clica no seletor de organization e escolhe "Startup XYZ"
And envia POST /auth/switch-organization com organization_id da "Startup XYZ"
Then o sistema valida que usuário é membro da "Startup XYZ"
And gera novo JWT com organization_id = "Startup XYZ"
And role pode ser diferente (era admin na ABC, é user na XYZ)
And retorna novos tokens
And o dashboard recarrega dados da "Startup XYZ"
```

**Regras de Negócio**:
- **RN-MT-01**: Usuário sem nenhuma organization não pode fazer login (deve criar organization primeiro)
- **RN-MT-02**: Token temporário expira em 15 minutos
- **RN-MT-03**: Token temporário só permite acessar endpoint /auth/select-organization
- **RN-MT-04**: Usuário só pode selecionar organizations das quais é membro
- **RN-MT-05**: Ao trocar organization, role pode mudar (admin em uma, user em outra)
- **RN-MT-06**: JWT final sempre contém organization_id + role específica daquela organization

---

## 3. Ciclo de Vida da Organization

### 3.1 Criação de Usuários e Convites

**Fora do Escopo desta Spec**

A criação de usuários, sistema de invites e onboarding estão documentados em specs separadas:
- `specs/functional/user-management.md` - Criação de usuários, convites (cenários A e B), onboarding
- `specs/functional/auth.md` - Autenticação, JWT, OAuth2

Esta spec foca apenas na estrutura de Organizations e isolamento multi-tenant.

### 3.2 Suspensão e Cancelamento

**Fora do Escopo desta Spec**

Suspensão por falta de pagamento, cancelamento voluntário e políticas de retenção estão documentados em:
- `specs/functional/subscription.md` - Planos, pagamentos, suspensão, cancelamento

Esta spec define apenas o campo `status` na tabela `organizations` (active, suspended, canceled).

---

## 4. Isolamento de Dados

### 4.1 Estrutura de Tabelas

**Tabelas Globais (SEM organization_id)**:

```sql
-- Tabela de Empresas/Organizações (raiz da organization)
CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,              -- "Empresa ABC"
    status VARCHAR(50) NOT NULL,             -- active, suspended, canceled
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

-- Tabela de Usuários (GLOBAL - sem organization_id)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,      -- Email ÚNICO no sistema inteiro
    password_hash VARCHAR(255) NOT NULL,
    created_at BIGINT NOT NULL,
    deleted_at BIGINT
    -- REMOVIDO: name, avatar_url (movidos para user_accounts)
);

-- Tabela de Dados da Conta (1:1 com users)
CREATE TABLE user_accounts (
    id UUID PRIMARY KEY,
    user_id UUID UNIQUE NOT NULL REFERENCES users(id),  -- 1:1 relationship
    full_name VARCHAR(255),
    avatar_url VARCHAR(500),
    phone VARCHAR(50),
    locale VARCHAR(10) DEFAULT 'pt-BR',
    timezone VARCHAR(50) DEFAULT 'America/Sao_Paulo',
    theme VARCHAR(20) DEFAULT 'light',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

CREATE INDEX idx_user_accounts_user_id ON user_accounts(user_id);

-- Tabela de Associação N:N (define role do user na organization)
CREATE TABLE organization_members (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(50) NOT NULL,               -- admin, member, guest
    invited_by UUID REFERENCES users(id),    -- quem convidou
    invited_at BIGINT NOT NULL,
    joined_at BIGINT,                        -- quando aceitou convite
    created_at BIGINT NOT NULL,
    deleted_at BIGINT,
    UNIQUE(organization_id, user_id)         -- User só pode estar 1x por organization
);

CREATE INDEX idx_organization_members_org ON organization_members(organization_id);
CREATE INDEX idx_organization_members_user ON organization_members(user_id);
```

> **Nota sobre Planos**: Campos relacionados a planos (plan, trial_ends_at, limites) serão adicionados na spec de Subscription (`specs/functional/subscription.md`). Esta spec define apenas a estrutura base de Organizations.

**Tabelas de Negócio (COM organization_id)**:

```sql
-- Assinaturas pertencem a uma organization
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at BIGINT NOT NULL,
    deleted_at BIGINT
);

CREATE INDEX idx_subscriptions_organization_id ON subscriptions(organization_id);

-- Pagamentos pertencem a uma organization
CREATE TABLE payments (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_payments_organization_id ON payments(organization_id);
```

### 4.2 Regras de Isolamento

**Princípios de Isolamento**:
- ✅ **Middleware extrai organization_id do JWT** (validado na autenticação)
- ✅ **Repositories SEMPRE filtram por organization_id** (explícito no código)
- ✅ **Índices em organization_id** garantem performance
- ✅ **Testes automatizados** validam isolamento
- ✅ Email é ÚNICO no sistema inteiro (não se repete)
- ✅ IDs são UUIDs globais (não há colisão)
- ✅ Usuário pode pertencer a múltiplas organizations com roles diferentes

**Por que NÃO usar RLS (Row-Level Security)?**
- ❌ Complexidade adicional (policies, SET statements)
- ❌ Mais difícil de debugar (isolamento "invisível")
- ❌ Performance overhead (avaliação de policies)
- ✅ **Filtro explícito no código é mais claro e testável**

**Exemplos de Queries Conceituais**:

**Buscar Subscription (Tabela com organization_id)**:
```sql
-- Query SEMPRE filtra por organization_id
SELECT * FROM subscriptions
WHERE id = :subscription_id
  AND organization_id = :organization_id_from_jwt  -- Isolamento
  AND deleted_at IS NULL;
```

**Listar Subscriptions (Tabela com organization_id)**:
```sql
-- TODAS as queries em tabelas de negócio filtram por organization_id
SELECT * FROM subscriptions
WHERE organization_id = :organization_id_from_jwt  -- Isolamento
  AND deleted_at IS NULL;
```

**Buscar User por Email (Tabela GLOBAL - sem organization_id)**:
```sql
-- Query NÃO filtra por organization (tabela global)
SELECT * FROM users
WHERE email = :email
  AND deleted_at IS NULL;
```

**Listar Organizations de um User (Cross-organization query)**:
```sql
-- Query que atravessa organizations para listar todas do usuário
SELECT o.*, om.role
FROM organizations o
JOIN organization_members om ON om.organization_id = o.id
WHERE om.user_id = :user_id
  AND om.deleted_at IS NULL
  AND o.deleted_at IS NULL;
```

**Validar Acesso do User à Organization**:
```sql
-- Valida que usuário é membro da organization antes de permitir acesso
SELECT COUNT(*) FROM organization_members
WHERE user_id = :user_id
  AND organization_id = :organization_id
  AND deleted_at IS NULL;

-- Se COUNT = 0, usuário NÃO é membro (erro 403 Forbidden)
-- Se COUNT = 1, usuário É membro (permite acesso)
```

**Nota**: A implementação técnica dessas queries está em `specs/technical/multi-tenancy-implementation.md`

---

## 5. Configurações por Organization

### 5.1 Organization Settings

Cada organization pode personalizar (armazenado em `organizations` ou `organization_settings`):

```json
{
  "organization_id": "uuid-abc",
  "branding": {
    "logo_url": "https://cdn.xyz.com/logo.png",
    "primary_color": "#3B82F6",
    "company_name": "Empresa ABC"
  },
  "features": {
    "max_users": 50,
    "modules": ["subscriptions", "reports"],
    "integrations": ["stripe", "sendgrid"]
  },
  "locale": {
    "language": "pt-BR",
    "timezone": "America/Sao_Paulo",
    "currency": "BRL",
    "date_format": "DD/MM/YYYY"
  },
  "notifications": {
    "email_sender": "noreply@empresaabc.com",
    "slack_webhook": "https://hooks.slack.com/..."
  }
}
```

**UI**: Admin pode alterar logo, cores, idioma no painel de configurações.

---

## 6. Segurança Multi-Tenant

### 6.1 Checklist de Segurança

**Obrigatório em TODO código**:
- ✅ Middleware SEMPRE valida JWT e extrai organization_id + user_id
- ✅ Context SEMPRE contém organization_id e user_id
- ✅ Repositories de tabelas de negócio SEMPRE filtram por organization_id (explicitamente)
- ✅ Validar que user é membro da organization antes de gerar JWT
- ✅ Testes automatizados validam isolamento entre organizations
- ✅ Code review verifica queries sem filtro de organization_id

**Anti-Patterns Proibidos**:
- ❌ Query em tabela de negócio sem `WHERE organization_id = ?`
- ❌ Usar organization_id do request body (só do JWT)
- ❌ Hardcoded organization_id
- ❌ Admin de uma organization acessar dados de outra
- ❌ Gerar JWT sem validar se user pertence à organization
- ❌ Adicionar organization_id na tabela users (users são globais)
- ❌ Usar `.Find()` sem filtro de organization em tabelas de negócio

### 6.2 Cenários de Teste de Isolamento

**Teste 1: Isolamento entre Organizations**

```gherkin
Given existem 2 organizations: "Organization A" e "Organization B"
And existe um usuário "joao@email.com"
And o usuário é membro apenas da "Organization A" (role: admin)
And a Organization A tem 1 subscription: "Sub A"
And a Organization B tem 1 subscription: "Sub B"
When o usuário faz login na Organization A
And tenta listar subscriptions
Then o sistema retorna apenas "Sub A"
And NÃO retorna "Sub B" (isolamento funcionando)

When o usuário tenta acessar diretamente a subscription "Sub B" por ID
Then o sistema retorna 404 Not Found (como se não existisse)
And NÃO expõe que a subscription existe em outra organization
```

**Teste 2: Usuário com múltiplas Organizations**

```gherkin
Given existem 2 organizations: "Organization A" e "Organization B"
And existe um usuário "joao@email.com"
And o usuário é membro de ambas as organizations:
  - Organization A com role "admin"
  - Organization B com role "member"
When o usuário faz login
Then o sistema retorna lista de 2 organizations
And mostra role específica de cada uma:
  - Organization A: admin
  - Organization B: member
When o usuário seleciona Organization A
Then o JWT contém organization_id = "Organization A"
And o JWT contém role = "admin"
And o usuário tem permissões de admin apenas na Organization A
When o usuário troca para Organization B (switch)
Then o novo JWT contém organization_id = "Organization B"
And o novo JWT contém role = "member"
And o usuário perde permissões de admin (agora é member)
```

**Teste 3: Tentativa de Acesso não Autorizado**

```gherkin
Given um usuário "attacker@email.com" membro apenas da Organization A
And existe Organization B com subscription "Sub B"
When o atacante tenta acessar diretamente:
  GET /api/subscriptions/{id_da_sub_B}
  Header: Authorization: Bearer {jwt_com_organization_a}
Then o sistema extrai organization_id = "Organization A" do JWT
And filtra query com WHERE organization_id = "Organization A"
And NÃO encontra a subscription (está na Organization B)
And retorna 404 Not Found
And NÃO expõe que a subscription existe
```

**Teste 4: Validação de Membership ao Trocar Organization**

```gherkin
Given um usuário membro apenas da Organization A
When ele tenta trocar para Organization B (da qual NÃO é membro)
And envia POST /auth/switch-organization com organization_id da B
Then o sistema valida membership
And NÃO encontra registro em organization_members
And retorna 403 Forbidden
And mensagem: "Você não é membro desta organização"
And NÃO permite a troca
```

### 6.3 Auditoria

**Logs incluem organization_id**:
```json
{
  "timestamp": "2025-11-05T10:30:00Z",
  "organization_id": "uuid-abc",
  "organization_name": "Empresa ABC",
  "user_id": "uuid-123",
  "action": "user.created",
  "resource_id": "uuid-456",
  "ip": "192.168.1.100"
}
```

---

## 7. Escalabilidade

### 7.1 Vantagens do Modelo Shared Database

**Zero Overhead de Provisionamento**:
- ✅ Nova organization = INSERT em `organizations` (< 1s)
- ✅ Sem criação de schema, database, servidor
- ✅ Onboarding 100% self-service
- ✅ Pode ter 10.000+ organizations no mesmo DB

**Escalabilidade Horizontal**:
- ✅ Adiciona mais servidores de aplicação
- ✅ Connection pooling compartilhado
- ✅ Cache compartilhado (Redis)
- ✅ Sharding por organization_id se necessário

**Performance**:
- ✅ Índices em `organization_id` garantem queries rápidas
- ✅ Queries pequenas (filtradas por organization)
- ✅ Particionamento por organization_id se tabela muito grande

### 7.2 Quando Sharding é Necessário

**Sinal**: > 1M organizations OU > 10TB de dados

**Estratégia**:
```
Database 1: organizations com organization_id hash % 3 == 0
Database 2: organizations com organization_id hash % 3 == 1
Database 3: organizations com organization_id hash % 3 == 2

Middleware roteia request para o shard correto baseado em organization_id
```

**Para AvantPro**: Single database é suficiente para 100K+ organizations

---

## 8. Comparação com Alternativas

### 8.1 Por Que NÃO Schema-per-Organization?

❌ Provisionamento lento (criar schema, executar migrations)
❌ Limite de ~1000 schemas no PostgreSQL
❌ Complexidade de migrations (rodar em N schemas)
❌ Backup/restore complexo

✅ **Shared Schema é melhor para**:
- SaaS self-service (Netflix, Spotify, Slack, GitHub)
- Rápido onboarding
- Muitas organizations pequenas/médias

### 8.2 Por Que NÃO Database-per-Organization?

❌ Overhead de infraestrutura (N databases)
❌ Custo proporcional a número de organizations
❌ Provisionamento manual
❌ Difícil fazer queries cross-organization (analytics)

---

## 9. Status Atual

**Implementado**:
- ❌ Nenhuma funcionalidade multi-tenant ainda

**Componentes Principais**:

1. **Domain Layer**:
   - Entidades: `Organization`, `OrganizationMember`, `UserAccount`
   - Repositories (interfaces): `OrganizationRepository`, `OrganizationMemberRepository`, `UserAccountRepository`

2. **Database Layer**:
   - Tabela `organizations` (name, status)
   - Tabela `organization_members` (N:N entre users e organizations)
   - Tabela `user_accounts` (1:1 com users - dados pessoais)
   - Tabela `users` simplificada (apenas email, password_hash)

3. **Application Layer**:
   - Middleware `OrganizationFromJWT` (extrai organization_id e valida membership)
   - Repositories filtram por organization_id (isolamento)
   - Endpoints de autenticação multi-organization

**Specs Relacionadas**:
- `specs/functional/auth.md` - Autenticação, JWT, OAuth2, login multi-organization
- `specs/functional/user-management.md` - Signup, convites, onboarding
- `specs/functional/subscription.md` - Planos, trials, limites, pagamentos, suspensão
