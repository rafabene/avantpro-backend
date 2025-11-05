# Multi-Tenancy - Requisitos Funcionais

**Versão**: 3.2
**Data**: 05/11/2025
**Changelog**:
- v3.0: Adicionado fluxo de login em 2 etapas (temp_token + select-tenant)
- v3.1: Removido RLS (Row-Level Security) - filtro explícito no código é preferível
- v3.2: Removida Fase 6 (Observabilidade) - não é escopo de multi-tenancy

---

## 1. Visão Geral

**AvantPro é um sistema SaaS multi-tenant** estilo Netflix/Spotify onde todos os clientes usam o mesmo domínio e compartilham a mesma infraestrutura, operando de forma isolada através de dados segregados.

### 1.1 Modelo de Multi-Tenancy

**Tipo**: **Shared Database + Shared Schema** (Row-Level Isolation)
- **Um único domínio**: `app.avantpro.com.br`
- **Um único banco de dados**: PostgreSQL compartilhado
- **Um único schema**: Todas as tabelas têm coluna `tenant_id`
- **Zero configuração de infraestrutura**: Cliente se registra e começa a usar
- **Escalabilidade horizontal**: Adiciona servidores, não databases

### 1.2 Conceitos

**Tenant (Empresa/Organização)**
- Representa uma empresa/cliente usando o sistema
- Exemplo: "Empresa ABC", "Startup XYZ"
- Cada tenant tem: assinaturas, configurações isoladas
- Identificado por UUID único
- **É a raiz do isolamento de dados**

**Usuário (Global)**
- Entidade **GLOBAL**, não pertence a nenhum tenant
- Identificado por email único no sistema inteiro
- **Pode pertencer a múltiplos tenants simultaneamente**
- Autenticação: email + senha (única para todos os tenants)

**TenantMember (Associação Usuário-Tenant)**
- Relacionamento N:N entre User e Tenant
- Define a **role específica** do usuário naquele tenant
- Exemplo: João pode ser:
  - Admin na Empresa ABC (tenant A)
  - User na Startup XYZ (tenant B)
  - Guest na Consultoria (tenant C)

**Isolamento por Dados**
- Todas as tabelas de negócio têm coluna `tenant_id UUID NOT NULL`
- Repositories SEMPRE filtram queries com `WHERE tenant_id = ?` (explícito no código)
- Middleware valida tenant_id do JWT antes de cada request
- Testes automatizados garantem isolamento entre tenants

---

## 2. Identificação de Tenant

### 2.1 Método de Identificação

**Domínio Único + Autenticação + Seleção de Tenant**
```
Todos acessam: https://app.avantpro.com.br

Login do usuário NÃO determina automaticamente o tenant:
- joao@email.com → pode pertencer a múltiplos tenants
- Usuário escolhe qual tenant quer acessar após login
```

**Fluxo de Login (2 etapas)**:

**Etapa 1: Autenticação (POST /auth/login)**
```
1. Usuário envia email + senha para `POST /auth/login`
2. Sistema valida credenciais
3. Sistema busca todos os tenants do usuário:
   SELECT t.*, tm.role
   FROM tenants t
   JOIN tenant_members tm ON tm.tenant_id = t.id
   WHERE tm.user_id = ?

4. Caso A - Usuário tem 1 tenant apenas:
   → Retorna JWT final (com tenant_id) + refresh_token
   → Usuário vai direto para dashboard

5. Caso B - Usuário tem múltiplos tenants:
   → Retorna token temporário + lista de tenants
   → Frontend mostra tela: "Selecione a organização"
```

**Etapa 2: Seleção de Tenant (POST /auth/select-tenant)** - apenas se múltiplos tenants
```
1. Frontend envia: { tenant_id } + temp_token no header
2. Sistema valida temp_token
3. Sistema valida que user é membro do tenant escolhido
4. Sistema gera JWT final (com tenant_id) + refresh_token
5. Usuário acessa dashboard do tenant escolhido
```

**Todas as requests subsequentes usam `tenant_id` do JWT**

### 2.2 Estrutura dos Tokens

**Token Temporário** (apenas para seleção de tenant):
```json
{
  "sub": "user-uuid-123",
  "email": "joao@email.com",
  "type": "tenant_selection",
  "exp": 1699123456  // 15 minutos
}
```

**JWT Final** (access_token - após selecionar tenant):
```json
{
  "sub": "user-uuid-123",
  "email": "joao@email.com",
  "tenant_id": "tenant-uuid-abc",
  "tenant_name": "Empresa ABC",
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
  "tenant_id": "tenant-uuid-abc",
  "type": "refresh",
  "exp": 1699728256  // 7 dias
}
```

**Middleware extrai dados do JWT**:
```go
ctx = context.WithValue(ctx, "user_id", claims.Sub)
ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)
ctx = context.WithValue(ctx, "role", claims.Role)  // role específica neste tenant
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

**Response A** (1 tenant apenas):
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "tenant": {
    "id": "uuid-abc",
    "name": "Empresa ABC",
    "slug": "empresa-abc",
    "role": "admin"
  }
}
```

**Response B** (múltiplos tenants):
```json
{
  "requires_tenant_selection": true,
  "temp_token": "eyJhbGc...",
  "tenants": [
    {
      "id": "uuid-abc",
      "name": "Empresa ABC",
      "slug": "empresa-abc",
      "role": "admin"
    },
    {
      "id": "uuid-xyz",
      "name": "Startup XYZ",
      "slug": "startup-xyz",
      "role": "user"
    }
  ]
}
```

---

**POST /auth/select-tenant** - Seleção de tenant (requer temp_token)

**Request**:
```http
POST /auth/select-tenant
Authorization: Bearer <temp_token>
Content-Type: application/json

{
  "tenant_id": "uuid-abc"
}
```

**Response**:
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "tenant": {
    "id": "uuid-abc",
    "name": "Empresa ABC",
    "slug": "empresa-abc",
    "role": "admin"
  }
}
```

---

**POST /auth/switch-tenant** - Trocar de organização (requer access_token)

**Request**:
```http
POST /auth/switch-tenant
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "tenant_id": "uuid-xyz"
}
```

**Response**:
```json
{
  "access_token": "eyJhbGc...",  // novo JWT com tenant_id diferente
  "refresh_token": "...",
  "tenant": {
    "id": "uuid-xyz",
    "name": "Startup XYZ",
    "slug": "startup-xyz",
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

    // 2. Buscar tenants do usuário
    members, err := s.tenantMemberRepo.FindByUserID(ctx, user.ID)
    if err != nil {
        return nil, err
    }

    if len(members) == 0 {
        return nil, errors.New("user_has_no_tenants")
    }

    // 3. Caso 1 tenant: gerar JWT final direto
    if len(members) == 1 {
        member := members[0]
        accessToken, _ := s.generateAccessToken(user, member.TenantID, member.Role)
        refreshToken, _ := s.generateRefreshToken(user, member.TenantID)

        return &LoginResponse{
            AccessToken:  accessToken,
            RefreshToken: refreshToken,
            Tenant:       toTenantInfo(member),
        }, nil
    }

    // 4. Múltiplos tenants: gerar token temporário
    tempToken, _ := s.generateTempToken(user)

    return &LoginResponse{
        RequiresTenantSelection: true,
        TempToken:               tempToken,
        Tenants:                 toTenantInfoList(members),
    }, nil
}

// AuthService.SelectTenant - Seleção de tenant
func (s *AuthService) SelectTenant(ctx context.Context, tempToken, tenantID string) (*TokenResponse, error) {
    // 1. Validar temp_token
    claims, err := s.validateTempToken(tempToken)
    if err != nil {
        return nil, errors.New("invalid_temp_token")
    }

    // 2. Validar que user é membro do tenant
    member, err := s.tenantMemberRepo.FindByUserAndTenant(ctx, claims.Sub, tenantID)
    if err != nil || member == nil {
        return nil, errors.New("user_not_member_of_tenant")
    }

    // 3. Gerar JWT final
    user, _ := s.userRepo.FindByID(ctx, claims.Sub)
    accessToken, _ := s.generateAccessToken(user, tenantID, member.Role)
    refreshToken, _ := s.generateRefreshToken(user, tenantID)

    return &TokenResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        Tenant:       toTenantInfo(member),
    }, nil
}

// AuthService.SwitchTenant - Trocar de organização
func (s *AuthService) SwitchTenant(ctx context.Context, userID, newTenantID string) (*TokenResponse, error) {
    // 1. Validar que user é membro do novo tenant
    member, err := s.tenantMemberRepo.FindByUserAndTenant(ctx, userID, newTenantID)
    if err != nil || member == nil {
        return nil, errors.New("user_not_member_of_tenant")
    }

    // 2. Gerar novo JWT
    user, _ := s.userRepo.FindByID(ctx, userID)
    accessToken, _ := s.generateAccessToken(user, newTenantID, member.Role)
    refreshToken, _ := s.generateRefreshToken(user, newTenantID)

    return &TokenResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        Tenant:       toTenantInfo(member),
    }, nil
}
```

**Frontend Flow**:
```javascript
// Login
async function login(email, password) {
  const response = await api.post('/auth/login', { email, password })

  if (response.requires_tenant_selection) {
    // Mostrar modal de seleção de tenant
    showTenantSelector(response.tenants, response.temp_token)
  } else {
    // Login direto (1 tenant)
    storeTokens(response.access_token, response.refresh_token)
    setCurrentTenant(response.tenant)
    redirectToDashboard()
  }
}

// Selecionar tenant
async function selectTenant(tenantId, tempToken) {
  const response = await api.post('/auth/select-tenant',
    { tenant_id: tenantId },
    { headers: { Authorization: `Bearer ${tempToken}` }}
  )

  storeTokens(response.access_token, response.refresh_token)
  setCurrentTenant(response.tenant)
  redirectToDashboard()
}

// Trocar de tenant
async function switchTenant(newTenantId) {
  const response = await api.post('/auth/switch-tenant',
    { tenant_id: newTenantId }
  )

  storeTokens(response.access_token, response.refresh_token)
  setCurrentTenant(response.tenant)
  window.location.reload() // Recarregar dados do novo tenant
}
```

---

## 3. Ciclo de Vida do Tenant

### 3.1 Criação (Self-Service)

**UC-01: Sign Up Automático**

**Ator**: Novo cliente (qualquer pessoa)

**Fluxo**:
1. Usuário acessa `app.avantpro.com.br/signup`
2. Preenche formulário:
   - Nome da organização (ex: "Empresa ABC")
   - Email pessoal (ex: "joao@empresaabc.com")
   - Nome completo
   - Senha
3. Sistema automaticamente:
   - Cria tenant com UUID único
   - Cria primeiro usuário como admin do tenant
   - Aplica plano trial (14 dias)
   - Envia email de confirmação
4. Usuário pode fazer login imediatamente
5. **Zero configuração manual ou de infraestrutura**

**Tempo**: < 2 segundos

**Pós-condições**:
- Tenant ativo no banco (1 row na tabela `tenants`)
- Usuário criado (1 row na tabela `users` - sem tenant_id)
- Associação criada (1 row na tabela `tenant_members` com role='admin')
- Trial iniciado
- Pronto para convidar mais usuários e criar assinaturas

### 3.2 Convite de Usuários

**UC-02: Admin Convida Usuário**

**Cenário A**: Email **NÃO existe** no sistema
1. Admin (do tenant ABC) convida `maria@email.com` com role='user'
2. Sistema cria convite com `tenant_id = ABC`, `role = 'user'`
3. Maria recebe email com link de convite
4. Maria clica, preenche nome e cria senha
5. Sistema:
   - Cria usuário na tabela `users` (sem tenant_id)
   - Cria associação na tabela `tenant_members` (tenant_id=ABC, role='user')
6. Maria faz login e vê apenas dados do tenant ABC

**Cenário B**: Email **JÁ existe** no sistema
1. Admin (do tenant ABC) convida `joao@email.com` com role='user'
2. Sistema verifica que João já existe (tem conta)
3. João recebe email: "Você foi convidado para Empresa ABC"
4. João faz login com credenciais existentes
5. Sistema mostra tela: "Selecione organização" (agora tem 2+ tenants)
6. Sistema cria apenas associação na `tenant_members` (tenant_id=ABC, role='user')
7. **João pode alternar entre seus tenants**

### 3.3 Suspensão/Cancelamento

**UC-03: Suspender por Falta de Pagamento**

**Fluxo**:
1. Sistema detecta pagamento vencido > 5 dias
2. Marca tenant como `status = 'suspended'`
3. Próximo login retorna erro: "Assinatura suspensa, atualize pagamento"
4. Dados preservados (não deletados)
5. Ao pagar, `status = 'active'` → acesso restaurado

**UC-04: Cancelamento Voluntário**

**Fluxo**:
1. Admin solicita cancelamento
2. `status = 'canceled'`, `canceled_at = NOW()`
3. Período de retenção: 30 dias (soft delete)
4. Após 30 dias: `deleted_at = NOW()` (hard delete via job)
5. Dados anonimizados/removidos (LGPD/GDPR)

---

## 4. Isolamento de Dados

### 4.1 Estrutura de Tabelas

**Tabelas Globais (SEM tenant_id)**:

```sql
-- Tabela de Empresas/Organizações (raiz do tenant)
CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,              -- "Empresa ABC"
    slug VARCHAR(100) UNIQUE NOT NULL,       -- "empresa-abc"
    status VARCHAR(50) NOT NULL,             -- active, suspended, canceled
    plan VARCHAR(50) NOT NULL,               -- trial, basic, premium
    trial_ends_at TIMESTAMP,
    created_at BIGINT NOT NULL,
    deleted_at BIGINT
);

-- Tabela de Usuários (GLOBAL - sem tenant_id)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,      -- Email ÚNICO no sistema inteiro
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at BIGINT NOT NULL,
    deleted_at BIGINT
);

-- Tabela de Associação N:N (define role do user no tenant)
CREATE TABLE tenant_members (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(50) NOT NULL,               -- admin, user, guest
    invited_by UUID REFERENCES users(id),    -- quem convidou
    invited_at BIGINT NOT NULL,
    joined_at BIGINT,                        -- quando aceitou convite
    created_at BIGINT NOT NULL,
    UNIQUE(tenant_id, user_id)               -- User só pode estar 1x por tenant
);

CREATE INDEX idx_tenant_members_tenant ON tenant_members(tenant_id);
CREATE INDEX idx_tenant_members_user ON tenant_members(user_id);
```

**Tabelas de Negócio (COM tenant_id)**:

```sql
-- Assinaturas pertencem a um tenant
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at BIGINT NOT NULL,
    deleted_at BIGINT
);

CREATE INDEX idx_subscriptions_tenant_id ON subscriptions(tenant_id);

-- Pagamentos pertencem a um tenant
CREATE TABLE payments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_payments_tenant_id ON payments(tenant_id);
```

### 4.2 Regras de Isolamento

**Princípios de Isolamento**:
- ✅ **Middleware extrai tenant_id do JWT** (validado na autenticação)
- ✅ **Repositories SEMPRE filtram por tenant_id** (explícito no código)
- ✅ **Índices em tenant_id** garantem performance
- ✅ **Testes automatizados** validam isolamento
- ✅ Email é ÚNICO no sistema inteiro (não se repete)
- ✅ IDs são UUIDs globais (não há colisão)
- ✅ Usuário pode pertencer a múltiplos tenants com roles diferentes

**Por que NÃO usar RLS (Row-Level Security)?**
- ❌ Complexidade adicional (policies, SET statements)
- ❌ Mais difícil de debugar (isolamento "invisível")
- ❌ Performance overhead (avaliação de policies)
- ✅ **Filtro explícito no código é mais claro e testável**

**Exemplos de Queries**:

```go
// Repository de Subscription (tem tenant_id)
func (r *SubscriptionRepository) FindByID(ctx context.Context, id string) (*Subscription, error) {
    tenantID := ctx.Value("tenant_id").(string)

    var sub SubscriptionModel
    // Query SEMPRE filtra por tenant_id (explícito e testável)
    err := r.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&sub).Error

    return toEntity(&sub), err
}

// Repository de Subscription - List
func (r *SubscriptionRepository) List(ctx context.Context) ([]Subscription, error) {
    tenantID := ctx.Value("tenant_id").(string)

    var subs []SubscriptionModel
    // TODAS as queries filtram por tenant_id
    err := r.db.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Find(&subs).Error

    return toEntities(subs), err
}

// Repository de User (GLOBAL - sem tenant_id)
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    var user UserModel
    // Query NÃO filtra por tenant (tabela global)
    err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
    return toEntity(&user), err
}

// Repository de TenantMember (busca tenants do user)
func (r *TenantMemberRepository) FindTenantsByUser(ctx context.Context, userID string) ([]TenantMember, error) {
    var members []TenantMemberModel
    // Query cross-tenant (sem filtro de tenant_id)
    err := r.db.Where("user_id = ?", userID).
        Preload("Tenant").
        Find(&members).Error
    return toEntities(members), err
}

// Service de Auth valida acesso ao tenant
func (s *AuthService) ValidateTenantAccess(ctx context.Context, userID, tenantID string) error {
    // Verifica se user é membro do tenant
    member, err := s.tenantMemberRepo.FindByUserAndTenant(ctx, userID, tenantID)
    if err != nil || member == nil {
        return errors.New("user not member of tenant")
    }
    return nil
}
```

---

## 5. Experiência do Usuário

### 5.1 Fluxo Completo (Novo Cliente)

**Dia 1 - Sign Up**:
```
1. João acessa: https://app.avantpro.com.br
2. Clica em "Começar Gratuitamente"
3. Preenche:
   - Organização: "Startup XYZ"
   - Email: joao@xyz.com
   - Nome: João Silva
   - Senha: *******
4. Clica "Criar Conta"
5. ✅ Conta criada em 2 segundos
6. Já está logado e vê dashboard vazio
7. Tutorial: "Convide sua equipe", "Crie primeira assinatura"
```

**Dia 2 - Convite de Equipe**:
```
1. João convida maria@xyz.com
2. Maria recebe email: "João te convidou para Startup XYZ"
3. Maria clica no link
4. Maria cria senha
5. ✅ Maria acessa sistema e vê dados da Startup XYZ
6. Maria NÃO vê dados de outros tenants
```

**Dia 14 - Trial Termina**:
```
1. Sistema envia email: "Trial termina em 3 dias"
2. João adiciona cartão de crédito
3. Sistema cobra automaticamente
4. Status continua "active"
```

### 5.2 Experiência Multi-Tenant

**UC: Usuário em Múltiplos Tenants**

Alguns usuários podem pertencer a múltiplos tenants (ex: consultor):

```
joao@consultor.com trabalha para:
- Tenant A - Empresa ABC (como admin)
- Tenant B - Startup XYZ (como user)
- Tenant C - Consultoria (como guest)

Login Flow (2 etapas):
1. João acessa app.avantpro.com.br
2. Digite email: joao@consultor.com + senha
3. Sistema retorna: requires_tenant_selection = true
4. Tela: "Selecione a organização"

   ┌─────────────────────────────────────┐
   │  Selecione uma organização          │
   ├─────────────────────────────────────┤
   │  [ ] Empresa ABC                    │
   │      Administrador                  │
   │                                     │
   │  [ ] Startup XYZ                    │
   │      Usuário                        │
   │                                     │
   │  [ ] Consultoria                    │
   │      Convidado                      │
   └─────────────────────────────────────┘

5. João seleciona "Empresa ABC"
6. POST /auth/select-tenant (com temp_token)
7. Sistema retorna JWT final com tenant_id = Empresa ABC
8. João acessa dashboard da Empresa ABC

Trocar de organização (já logado):
1. Menu superior: Dropdown com "Empresa ABC ▼"
2. João clica e vê lista de organizações
3. João seleciona "Startup XYZ"
4. POST /auth/switch-tenant
5. Sistema retorna novo JWT com tenant_id = Startup XYZ
6. Página recarrega com dados da Startup XYZ
7. Agora João é "Usuário" (não mais Admin)
```

**UC: Usuário Convidado para Novo Tenant**

```
Situação inicial:
- Maria já usa AvantPro na "Empresa ABC" (como admin)
- Pedro (da Startup XYZ) convida maria@email.com (como user)

Flow:
1. Maria recebe email: "Pedro convidou você para Startup XYZ"
2. Maria faz login normalmente (maria@email.com + senha)
3. Sistema agora retorna 2 tenants na lista:
   - Empresa ABC (Admin)
   - Startup XYZ (Usuário) ← NOVO
4. Maria pode escolher entre os 2
5. Maria acessa Startup XYZ como "Usuário"
6. Maria pode alternar entre tenants a qualquer momento
```

---

## 6. Configurações por Tenant

### 6.1 Tenant Settings

Cada tenant pode personalizar (armazenado em `tenants` ou `tenant_settings`):

```json
{
  "tenant_id": "uuid-abc",
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

## 7. Planos e Limites

### 7.1 Planos

**Trial** (14 dias grátis):
- 5 usuários
- Funcionalidades básicas
- Suporte via email

**Basic** (R$ 99/mês):
- 20 usuários
- Todas as funcionalidades
- Suporte via email

**Premium** (R$ 299/mês):
- Usuários ilimitados
- API access
- Suporte prioritário
- Customizações

### 7.2 Enforcement de Limites

```go
// Ao convidar/adicionar membro ao tenant
func (s *TenantMemberService) InviteUser(ctx context.Context, input InviteUserInput) error {
    tenantID := ctx.Value("tenant_id").(string)

    // Buscar tenant
    tenant := s.tenantRepo.FindByID(ctx, tenantID)

    // Contar membros atuais do tenant
    count := s.tenantMemberRepo.CountByTenant(ctx, tenantID)

    // Verificar limite do plano
    limit := getPlanLimit(tenant.Plan) // trial: 5, basic: 20, premium: unlimited

    if count >= limit {
        return errors.New("error.user_limit_reached")
    }

    // Convidar/criar associação...
}
```

---

## 8. Segurança Multi-Tenant

### 8.1 Checklist de Segurança

**Obrigatório em TODO código**:
- ✅ Middleware SEMPRE valida JWT e extrai tenant_id + user_id
- ✅ Context SEMPRE contém tenant_id e user_id
- ✅ Repositories de tabelas de negócio SEMPRE filtram por tenant_id (explicitamente)
- ✅ Validar que user é membro do tenant antes de gerar JWT
- ✅ Testes automatizados validam isolamento entre tenants
- ✅ Code review verifica queries sem filtro de tenant_id

**Anti-Patterns Proibidos**:
- ❌ Query em tabela de negócio sem `WHERE tenant_id = ?`
- ❌ Usar tenant_id do request body (só do JWT)
- ❌ Hardcoded tenant_id
- ❌ Admin de um tenant acessar dados de outro
- ❌ Gerar JWT sem validar se user pertence ao tenant
- ❌ Adicionar tenant_id na tabela users (users são globais)
- ❌ Usar `.Find()` sem filtro de tenant em tabelas de negócio

### 8.2 Teste de Isolamento

```go
func TestTenantIsolation(t *testing.T) {
    // Criar 2 tenants
    tenant1 := createTenant("Tenant A")
    tenant2 := createTenant("Tenant B")

    // Criar usuário global
    user := createUser("joao@email.com")

    // Associar user ao tenant1 como admin
    createTenantMember(tenant1.ID, user.ID, "admin")

    // Criar subscriptions em cada tenant
    sub1 := createSubscription(tenant1.ID, "Sub A")
    sub2 := createSubscription(tenant2.ID, "Sub B")

    // Login como user no tenant A
    ctx := contextWithTenant(user.ID, tenant1.ID)

    // Tentar buscar subscriptions
    subs := subscriptionRepo.List(ctx)

    // DEVE retornar apenas sub1 (do tenant A)
    assert.Len(t, subs, 1)
    assert.Equal(t, sub1.ID, subs[0].ID)

    // Tentar acessar sub2 diretamente (do tenant B)
    found := subscriptionRepo.FindByID(ctx, sub2.ID)

    // DEVE retornar nil (não encontrado - isolamento funcionando)
    assert.Nil(t, found)
}

func TestMultiTenantUser(t *testing.T) {
    // Criar 2 tenants
    tenantA := createTenant("Tenant A")
    tenantB := createTenant("Tenant B")

    // Criar usuário global
    user := createUser("joao@email.com")

    // Associar user a ambos os tenants com roles diferentes
    createTenantMember(tenantA.ID, user.ID, "admin")
    createTenantMember(tenantB.ID, user.ID, "user")

    // Buscar tenants do user
    tenants := tenantMemberRepo.FindTenantsByUser(user.ID)

    // DEVE retornar 2 tenants
    assert.Len(t, tenants, 2)
    assert.Equal(t, "admin", tenants[0].Role) // Tenant A
    assert.Equal(t, "user", tenants[1].Role)  // Tenant B
}
```

### 8.3 Auditoria

**Logs incluem tenant_id**:
```json
{
  "timestamp": "2025-11-05T10:30:00Z",
  "tenant_id": "uuid-abc",
  "tenant_name": "Empresa ABC",
  "user_id": "uuid-123",
  "action": "user.created",
  "resource_id": "uuid-456",
  "ip": "192.168.1.100"
}
```

---

## 9. Escalabilidade

### 9.1 Vantagens do Modelo Shared Database

**Zero Overhead de Provisionamento**:
- ✅ Novo tenant = INSERT em `tenants` (< 1s)
- ✅ Sem criação de schema, database, servidor
- ✅ Onboarding 100% self-service
- ✅ Pode ter 10.000+ tenants no mesmo DB

**Escalabilidade Horizontal**:
- ✅ Adiciona mais servidores de aplicação
- ✅ Connection pooling compartilhado
- ✅ Cache compartilhado (Redis)
- ✅ Sharding por tenant_id se necessário

**Performance**:
- ✅ Índices em `tenant_id` garantem queries rápidas
- ✅ Queries pequenas (filtradas por tenant)
- ✅ Particionamento por tenant_id se tabela muito grande

### 9.2 Quando Sharding é Necessário

**Sinal**: > 1M tenants OU > 10TB de dados

**Estratégia**:
```
Database 1: tenants com tenant_id hash % 3 == 0
Database 2: tenants com tenant_id hash % 3 == 1
Database 3: tenants com tenant_id hash % 3 == 2

Middleware roteia request para o shard correto baseado em tenant_id
```

**Para AvantPro**: Single database é suficiente para 100K+ tenants

---

## 10. Comparação com Alternativas

### 10.1 Por Que NÃO Schema-per-Tenant?

❌ Provisionamento lento (criar schema, executar migrations)
❌ Limite de ~1000 schemas no PostgreSQL
❌ Complexidade de migrations (rodar em N schemas)
❌ Backup/restore complexo

✅ **Shared Schema é melhor para**:
- SaaS self-service (Netflix, Spotify, Slack)
- Rápido onboarding
- Muitos tenants pequenos/médios

### 10.2 Por Que NÃO Database-per-Tenant?

❌ Overhead de infraestrutura (N databases)
❌ Custo proporcional a número de tenants
❌ Provisionamento manual
❌ Difícil fazer queries cross-tenant (analytics)

---

## 11. Roadmap de Implementação

### Fase 1: Fundação ✅ Crítico
- [ ] Entidade `Tenant` no domínio
- [ ] Entidade `TenantMember` no domínio
- [ ] Migration `create_tenants.up.sql`
- [ ] Migration `create_tenant_members.up.sql`
- [ ] **REMOVER** `tenant_id` da tabela `users` (users são globais)
- [ ] Adicionar `tenant_id` em tabelas de negócio (subscriptions, payments, etc)
- [ ] Middleware `TenantFromJWT` extrai tenant_id e user_id
- [ ] Repositories de negócio filtram por tenant_id
- [ ] Testes de isolamento

### Fase 2: Autenticação Multi-Tenant
- [ ] Service: `generateTempToken()` - token temporário de seleção
- [ ] Service: `generateAccessToken(user, tenantID, role)` - JWT final
- [ ] Service: `validateTempToken()` - validação de temp_token
- [ ] Endpoint `POST /auth/login` (2 respostas: direto ou requires_tenant_selection)
- [ ] Endpoint `POST /auth/select-tenant` (recebe temp_token, retorna JWT final)
- [ ] Endpoint `POST /auth/switch-tenant` (troca de organização)
- [ ] Middleware: validar que user é membro do tenant (antes de cada request)
- [ ] Cache (Redis): membership validation (evitar query em cada request)

### Fase 3: Onboarding Self-Service
- [ ] Endpoint `POST /auth/signup` (cria tenant + user + tenant_member com role=admin)
- [ ] Email de boas-vindas
- [ ] Convite de usuários (cenários A e B)
- [ ] Endpoint `POST /tenants/:id/members/invite` (convidar user)
- [ ] Endpoint `POST /invitations/:token/accept` (aceitar convite)

### Fase 4: Planos e Limites
- [ ] Enforcement de limites por plano
- [ ] Trial de 14 dias
- [ ] Upgrade/downgrade de plano
- [ ] Suspensão por falta de pagamento

### Fase 5: Configurações por Tenant
- [ ] Tenant settings (branding, locale)
- [ ] Upload de logo
- [ ] Customização de cores
- [ ] Webhook URLs personalizadas por tenant

---

## Status Atual

**Implementado**:
- ❌ Nenhuma funcionalidade multi-tenant ainda

**Próximo Passo Crítico** (Fase 1):
1. ⚠️ **Criar entidades `Tenant` e `TenantMember` no domínio**
2. ⚠️ **Migrations para `tenants` e `tenant_members` tables**
3. ⚠️ **REMOVER `tenant_id` da tabela `users` (users são globais)**
4. ⚠️ **Adicionar `tenant_id` em tabelas de negócio (subscriptions, etc)**
5. ⚠️ **Repositories: `TenantRepository` e `TenantMemberRepository`**
6. ⚠️ **Atualizar repositories de negócio para filtrar por tenant_id**

**Depois (Fase 2)**:
7. ⚠️ **Service: métodos de geração de tokens (temp + final)**
8. ⚠️ **Endpoints: `/auth/login`, `/auth/select-tenant`, `/auth/switch-tenant`**
9. ⚠️ **Middleware `TenantFromJWT` (extrai tenant_id e valida membership)**
