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

### 2.4 Implementação do Fluxo de Login

**Service Layer**:
```go
// AuthService.Login - Autenticação inicial
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
    // 1. Validar credenciais
    user, err := s.userRepo.FindByEmail(ctx, email)
    if err != nil || !s.validatePassword(user, password) {
        return nil, errors.New("invalid_credentials")
    }

    // 2. Buscar organizations do usuário
    members, err := s.organizationMemberRepo.FindByUserID(ctx, user.ID)
    if err != nil {
        return nil, err
    }

    if len(members) == 0 {
        return nil, errors.New("user_has_no_organizations")
    }

    // 3. Caso 1 organization: gerar JWT final direto
    if len(members) == 1 {
        member := members[0]
        accessToken, _ := s.generateAccessToken(user, member.OrganizationID, member.Role)
        refreshToken, _ := s.generateRefreshToken(user, member.OrganizationID)

        return &LoginResponse{
            AccessToken:  accessToken,
            RefreshToken: refreshToken,
            Organization: toOrganizationInfo(member),
        }, nil
    }

    // 4. Múltiplas organizations: gerar token temporário
    tempToken, _ := s.generateTempToken(user)

    return &LoginResponse{
        RequiresOrganizationSelection: true,
        TempToken:                      tempToken,
        Organizations:                  toOrganizationInfoList(members),
    }, nil
}

// AuthService.SelectOrganization - Seleção de organization
func (s *AuthService) SelectOrganization(ctx context.Context, tempToken, organizationID string) (*TokenResponse, error) {
    // 1. Validar temp_token
    claims, err := s.validateTempToken(tempToken)
    if err != nil {
        return nil, errors.New("invalid_temp_token")
    }

    // 2. Validar que user é membro da organization
    member, err := s.organizationMemberRepo.FindByUserAndOrganization(ctx, claims.Sub, organizationID)
    if err != nil || member == nil {
        return nil, errors.New("user_not_member_of_organization")
    }

    // 3. Gerar JWT final
    user, _ := s.userRepo.FindByID(ctx, claims.Sub)
    accessToken, _ := s.generateAccessToken(user, organizationID, member.Role)
    refreshToken, _ := s.generateRefreshToken(user, organizationID)

    return &TokenResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        Organization: toOrganizationInfo(member),
    }, nil
}

// AuthService.SwitchOrganization - Trocar de organização
func (s *AuthService) SwitchOrganization(ctx context.Context, userID, newOrganizationID string) (*TokenResponse, error) {
    // 1. Validar que user é membro da nova organization
    member, err := s.organizationMemberRepo.FindByUserAndOrganization(ctx, userID, newOrganizationID)
    if err != nil || member == nil {
        return nil, errors.New("user_not_member_of_organization")
    }

    // 2. Gerar novo JWT
    user, _ := s.userRepo.FindByID(ctx, userID)
    accessToken, _ := s.generateAccessToken(user, newOrganizationID, member.Role)
    refreshToken, _ := s.generateRefreshToken(user, newOrganizationID)

    return &TokenResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        Organization: toOrganizationInfo(member),
    }, nil
}
```

**Frontend Flow**:
```javascript
// Login
async function login(email, password) {
  const response = await api.post('/auth/login', { email, password })

  if (response.requires_organization_selection) {
    // Mostrar modal de seleção de organization
    showOrganizationSelector(response.organizations, response.temp_token)
  } else {
    // Login direto (1 organization)
    storeTokens(response.access_token, response.refresh_token)
    setCurrentOrganization(response.organization)
    redirectToDashboard()
  }
}

// Selecionar organization
async function selectOrganization(organizationId, tempToken) {
  const response = await api.post('/auth/select-organization',
    { organization_id: organizationId },
    { headers: { Authorization: `Bearer ${tempToken}` }}
  )

  storeTokens(response.access_token, response.refresh_token)
  setCurrentOrganization(response.organization)
  redirectToDashboard()
}

// Trocar de organization
async function switchOrganization(newOrganizationId) {
  const response = await api.post('/auth/switch-organization',
    { organization_id: newOrganizationId }
  )

  storeTokens(response.access_token, response.refresh_token)
  setCurrentOrganization(response.organization)
  window.location.reload() // Recarregar dados da nova organization
}
```

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

**Exemplos de Queries**:

```go
// Repository de Subscription (tem organization_id)
func (r *SubscriptionRepository) FindByID(ctx context.Context, id string) (*Subscription, error) {
    organizationID := ctx.Value("organization_id").(string)

    var sub SubscriptionModel
    // Query SEMPRE filtra por organization_id (explícito e testável)
    err := r.db.Where("id = ? AND organization_id = ?", id, organizationID).First(&sub).Error

    return toEntity(&sub), err
}

// Repository de Subscription - List
func (r *SubscriptionRepository) List(ctx context.Context) ([]Subscription, error) {
    organizationID := ctx.Value("organization_id").(string)

    var subs []SubscriptionModel
    // TODAS as queries filtram por organization_id
    err := r.db.Where("organization_id = ? AND deleted_at IS NULL", organizationID).Find(&subs).Error

    return toEntities(subs), err
}

// Repository de User (GLOBAL - sem organization_id)
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    var user UserModel
    // Query NÃO filtra por organization (tabela global)
    err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
    return toEntity(&user), err
}

// Repository de OrganizationMember (busca organizations do user)
func (r *OrganizationMemberRepository) FindOrganizationsByUser(ctx context.Context, userID string) ([]OrganizationMember, error) {
    var members []OrganizationMemberModel
    // Query cross-organization (sem filtro de organization_id)
    err := r.db.Where("user_id = ?", userID).
        Preload("Organization").
        Find(&members).Error
    return toEntities(members), err
}

// Service de Auth valida acesso à organization
func (s *AuthService) ValidateOrganizationAccess(ctx context.Context, userID, organizationID string) error {
    // Verifica se user é membro da organization
    member, err := s.organizationMemberRepo.FindByUserAndOrganization(ctx, userID, organizationID)
    if err != nil || member == nil {
        return errors.New("user not member of organization")
    }
    return nil
}
```

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

### 6.2 Teste de Isolamento

```go
func TestOrganizationIsolation(t *testing.T) {
    // Criar 2 organizations
    org1 := createOrganization("Organization A")
    org2 := createOrganization("Organization B")

    // Criar usuário global
    user := createUser("joao@email.com")

    // Associar user à org1 como admin
    createOrganizationMember(org1.ID, user.ID, "admin")

    // Criar subscriptions em cada organization
    sub1 := createSubscription(org1.ID, "Sub A")
    sub2 := createSubscription(org2.ID, "Sub B")

    // Login como user na org A
    ctx := contextWithOrganization(user.ID, org1.ID)

    // Tentar buscar subscriptions
    subs := subscriptionRepo.List(ctx)

    // DEVE retornar apenas sub1 (da org A)
    assert.Len(t, subs, 1)
    assert.Equal(t, sub1.ID, subs[0].ID)

    // Tentar acessar sub2 diretamente (da org B)
    found := subscriptionRepo.FindByID(ctx, sub2.ID)

    // DEVE retornar nil (não encontrado - isolamento funcionando)
    assert.Nil(t, found)
}

func TestMultiOrganizationUser(t *testing.T) {
    // Criar 2 organizations
    orgA := createOrganization("Organization A")
    orgB := createOrganization("Organization B")

    // Criar usuário global
    user := createUser("joao@email.com")

    // Associar user a ambas as organizations com roles diferentes
    createOrganizationMember(orgA.ID, user.ID, "admin")
    createOrganizationMember(orgB.ID, user.ID, "member")

    // Buscar organizations do user
    orgs := organizationMemberRepo.FindOrganizationsByUser(user.ID)

    // DEVE retornar 2 organizations
    assert.Len(t, orgs, 2)
    assert.Equal(t, "admin", orgs[0].Role)  // Organization A
    assert.Equal(t, "member", orgs[1].Role) // Organization B
}
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
