# Registro de Usuários - Requisitos Funcionais

**Versão**: 2.2
**Data**: 05/11/2025
**Changelog**:
- v2.2: Removido slug de organizations (identificação apenas por UUID + name)
- v2.1: Removido campo nome completo do registro (apenas email + senha)
- v2.0: Reescrita completa com fluxo correto (verificação obrigatória, organization criada no primeiro login)

---

## 1. Visão Geral

O AvantPro segue um **fluxo de registro em etapas** onde a criação da conta é separada da criação da Organization:

### 1.1 Etapas do Registro Completo

```
1. Registro Inicial → 2. Verificação Email → 3. Primeiro Login → 4. Criar/Selecionar Organization
```

**Etapa 1 - Registro Inicial** (`POST /auth/register`):
- Usuário fornece: **apenas email + senha**
- Sistema cria User com status=inactive
- Sistema envia email de verificação

**Etapa 2 - Verificação de Email** (`POST /auth/verify-email`):
- **Obrigatória** - Conta permanece inativa até verificação
- Usuário clica no link do email
- Sistema ativa conta (status=active)

**Etapa 3 - Primeiro Login** (`POST /auth/login`):
- Usuário faz login com email/senha
- Sistema detecta que user não tem Organizations
- Sistema redireciona para criar Organization

**Etapa 4 - Criar Organization** (`POST /organizations`):
- Usuário fornece nome da empresa
- Sistema cria Organization + OrganizationMember (owner)
- Sistema retorna JWT final com organization_id

### 1.2 Fluxo Alternativo: Aceitar Convite

Usuário pode pular etapa 4 se receber convite:

```
1. Registro → 2. Verificação → 3. Aceitar Convite → 4. Login com Organization
```

---

## 2. Casos de Uso

### 2.1 UC-01: Registro Inicial (Criar Conta)

**Ator**: Usuário não cadastrado
**Objetivo**: Criar conta no sistema (SEM criar Organization ainda)

**Pré-Condições**:
- Email não cadastrado no sistema

**Fluxo Principal**:
1. Usuário acessa página de registro `/signup`
2. Usuário preenche formulário:
   - **Email** (obrigatório, único no sistema)
   - **Senha** (obrigatório, mínimo 8 caracteres)
3. Sistema valida dados:
   - Email válido e não cadastrado
   - Senha forte (mínimo 8 caracteres, 1 letra, 1 número)
4. Sistema cria:
   - Novo `User`:
     - Email + password hash
     - **Status**: `inactive` (conta não verificada)
     - EmailVerifiedAt: null
5. Sistema gera token de verificação (UUID único)
6. Sistema envia **email de verificação** com link:
   - `https://app.avantpro.com.br/verify-email?token=abc123`
   - Token expira em 24 horas
7. Sistema retorna resposta 201 Created:
   ```json
   {
     "user_id": "uuid-123",
     "email": "joao@email.com",
     "status": "inactive",
     "message": "Enviamos um email de verificação. Verifique sua caixa de entrada."
   }
   ```

**Pós-Condições**:
- Usuário criado com status=inactive
- Email de verificação enviado
- Usuário **NÃO pode fazer login** até verificar email

**Regras de Negócio**:
- **RN-01**: Email deve ser único no sistema inteiro
- **RN-02**: Conta inicia inativa e só ativa após verificação de email
- **RN-03**: Usuário não pode fazer login com conta inativa

---

### 2.2 UC-02: Verificação de Email

**Ator**: Usuário com conta inativa
**Objetivo**: Ativar conta verificando posse do email

**Pré-Condições**:
- User criado com status=inactive
- Token de verificação válido (não expirado)

**Fluxo Principal**:
1. Usuário clica no link do email recebido
2. Frontend extrai token da URL e chama `POST /auth/verify-email`
3. Sistema valida token:
   - Token existe no banco
   - Token não expirou (24 horas)
   - Email ainda não verificado
4. Sistema ativa conta:
   - User.Status: `inactive` → `active`
   - User.EmailVerifiedAt: timestamp atual
   - Marca token como usado (não pode reutilizar)
5. Sistema retorna tokens JWT temporários (apenas para completar cadastro):
   ```json
   {
     "access_token": "eyJhbGc...",
     "message": "Email verificado com sucesso!",
     "next_step": "create_organization"
   }
   ```
6. Frontend redireciona para:
   - Se user tem Organizations: dashboard
   - Se user não tem Organizations: criar organization

**Fluxos Alternativos**:

**2.2.1 - Token Expirado**:
- Sistema retorna erro 410 Gone
- Frontend oferece botão "Reenviar email de verificação"

**2.2.2 - Email Já Verificado**:
- Sistema retorna 200 OK com mensagem: "Email já verificado"
- Frontend redireciona para login

**Pós-Condições**:
- Usuário ativado (status=active)
- Email marcado como verificado
- Usuário pode fazer login

**Regras de Negócio**:
- **RN-04**: Token de verificação expira em 24 horas
- **RN-05**: Token é single-use (não pode reutilizar)
- **RN-06**: Usuário pode solicitar reenvio ilimitado do email
- **RN-07**: Conta inativa não pode fazer login

---

### 2.3 UC-03: Primeiro Login (Sem Organization)

**Ator**: Usuário com conta ativa mas sem Organization
**Objetivo**: Fazer login e ser redirecionado para criar Organization

**Pré-Condições**:
- User com status=active
- Email verificado
- User não pertence a nenhuma Organization

**Fluxo Principal**:
1. Usuário acessa `/login` e fornece email + senha
2. Sistema valida credenciais
3. Sistema verifica Organizations do usuário:
   ```sql
   SELECT COUNT(*) FROM organization_members
   WHERE user_id = ? AND deleted_at IS NULL
   ```
4. **Caso A - Nenhuma Organization**:
   - Sistema gera token temporário (tipo: "onboarding")
   - Token contém apenas user_id (SEM organization_id)
   - Sistema retorna:
   ```json
   {
     "access_token": "temp-token-eyJhbGc...",
     "token_type": "onboarding",
     "next_step": "create_organization",
     "message": "Crie sua organização para começar"
   }
   ```
   - Frontend redireciona para `/onboarding/create-organization`

5. **Caso B - Tem Organizations**:
   - Segue fluxo normal de multi-tenancy (spec multi-tenancy.md)
   - Se 1 organization: retorna JWT com organization_id
   - Se múltiplas: mostra seletor de organization

**Pós-Condições**:
- Usuário autenticado com token temporário
- Aguardando criação de Organization

**Regras de Negócio**:
- **RN-08**: Token de onboarding expira em 1 hora
- **RN-09**: Token de onboarding não permite acessar recursos de negócio (apenas criar organization)
- **RN-10**: Usuário sem organization não pode acessar dashboard

---

### 2.4 UC-04: Criar Organization (Primeira Vez)

**Ator**: Usuário autenticado com token de onboarding
**Objetivo**: Criar primeira Organization e se tornar owner

**Pré-Condições**:
- User autenticado com token de onboarding
- User não tem Organizations

**Fluxo Principal**:
1. Frontend exibe formulário `/onboarding/create-organization`:
   - Nome da empresa (obrigatório)
2. Usuário envia `POST /organizations` com token de onboarding
3. Sistema valida:
   - Token de onboarding válido
   - User não tem Organizations (previne duplicação)
   - Nome da empresa válido
4. Sistema cria **atomicamente**:
   - Nova `Organization`:
     - Name: nome fornecido
     - Status: active
     - TrialEndsAt: now() + 14 dias (se trial habilitado)
   - `OrganizationMember`:
     - UserID: user do token
     - OrganizationID: organization criada
     - Role: **owner** (primeiro membro sempre owner)
     - JoinedAt: now()
5. Sistema gera **JWT final** (com organization_id):
   ```json
   {
     "sub": "user-uuid-123",
     "email": "joao@email.com",
     "organization_id": "org-uuid-abc",
     "organization_name": "Empresa ABC",
     "role": "owner",
     "permissions": ["*:*"],
     "type": "access",
     "exp": 1699124356
   }
   ```
6. Sistema retorna:
   ```json
   {
     "access_token": "eyJhbGc...",
     "refresh_token": "...",
     "organization": {
       "id": "uuid-abc",
       "name": "Empresa ABC",
       "role": "owner",
       "trial_ends_at": 1699900800
     }
   }
   ```
7. Frontend redireciona para dashboard da organization

**Pós-Condições**:
- Organization criada
- Usuário é owner da organization
- JWT contém organization_id
- Trial iniciado (se configurado)

**Regras de Negócio**:
- **RN-11**: Primeiro membro sempre é owner
- **RN-12**: Organization inicia com trial gratuito de 14 dias (configurável)
- **RN-13**: Usuário só pode criar organization com token de onboarding

---

### 2.5 UC-05: Aceitar Convite

**Ator**: Usuário não cadastrado OU usuário cadastrado
**Objetivo**: Aceitar convite e se juntar a Organization existente

**Pré-Condições**:
- Convite válido (não expirado, status=pending)

**Fluxo Principal**:

**Caso A - Usuário Não Cadastrado**:

1. Admin envia convite para `maria@email.com`
2. Sistema cria registro de convite pendente
3. Sistema envia email com link: `https://app.avantpro.com.br/accept-invite?token=xyz`
4. Maria clica no link
5. Frontend exibe formulário:
   - Email: maria@email.com (readonly, extraído do token)
   - Senha (obrigatória)
   - Nome completo (opcional)
6. Maria envia `POST /auth/accept-invite`
7. Sistema cria **atomicamente**:
   - `User`:
     - Email + password hash
     - **Status**: `active` (convite já valida email)
     - EmailVerifiedAt: now() (considera email validado)
   - `UserAccount`:
     - FullName: nome fornecido ou null
   - `OrganizationMember`:
     - UserID: user criado
     - OrganizationID: do convite
     - Role: definida no convite (admin, member, guest)
     - JoinedAt: now()
   - Marca convite como aceito (status=accepted)
8. Sistema gera JWT com organization_id
9. Frontend redireciona para dashboard

**Caso B - Usuário Já Cadastrado**:

1. João (já tem conta) recebe convite para outra organization
2. João clica no link do convite
3. Frontend detecta que email já existe:
   - Solicita apenas senha (login)
4. João faz login com senha
5. Sistema valida:
   - Senha correta
   - João ainda não é membro da organization do convite
6. Sistema cria apenas:
   - `OrganizationMember` (adiciona João à organization)
   - Marca convite como aceito
7. Sistema gera JWT com nova organization_id
8. Frontend redireciona para dashboard da nova organization

**Fluxos Alternativos**:

**2.5.1 - Usuário Já é Membro**:
- Sistema retorna erro 409 Conflict
- Mensagem: "Você já é membro desta organização"

**2.5.2 - Convite Expirado**:
- Sistema retorna erro 410 Gone
- Frontend exibe: "Convite expirado. Solicite novo convite ao administrador."

**Pós-Condições**:
- User criado ou associado à organization
- Convite marcado como aceito
- Usuário pode acessar dashboard da organization

**Regras de Negócio**:
- **RN-15**: Aceitar convite valida email automaticamente (não precisa verificar)
- **RN-16**: Convite expira em 7 dias
- **RN-17**: User só pode ter 1 convite pendente por organization
- **RN-18**: Token de convite é single-use

---

## 3. Fluxos Detalhados

### 3.1 Fluxo Completo: Registro → Verificação → Criar Organization

```
┌─────────────────────────────────────────────────┐
│ ETAPA 1: Registro Inicial                       │
│ POST /auth/register                             │
└─────────────────────────────────────────────────┘
        ↓
Body: {
  "email": "joao@email.com",
  "password": "Senha123"
}
        ↓
┌─────────────────────────────────────────────────┐
│ Sistema cria User (status=inactive)             │
│ Sistema gera token de verificação               │
│ Sistema envia email de verificação              │
└─────────────────────────────────────────────────┘
        ↓
Response 201: {
  "user_id": "uuid-123",
  "status": "inactive",
  "message": "Verifique seu email"
}
        ↓
┌─────────────────────────────────────────────────┐
│ ETAPA 2: Verificação de Email                   │
│ POST /auth/verify-email                         │
└─────────────────────────────────────────────────┘
        ↓
Body: {
  "token": "verification-token-abc"
}
        ↓
┌─────────────────────────────────────────────────┐
│ Sistema valida token                            │
│ Sistema ativa User (status=active)              │
│ Sistema marca EmailVerifiedAt                   │
└─────────────────────────────────────────────────┘
        ↓
Response 200: {
  "message": "Email verificado!",
  "next_step": "login"
}
        ↓
┌─────────────────────────────────────────────────┐
│ ETAPA 3: Primeiro Login                         │
│ POST /auth/login                                │
└─────────────────────────────────────────────────┘
        ↓
Body: {
  "email": "joao@email.com",
  "password": "Senha123"
}
        ↓
┌─────────────────────────────────────────────────┐
│ Sistema valida credenciais                      │
│ Sistema busca Organizations do user             │
│ COUNT = 0 (nenhuma organization)                │
└─────────────────────────────────────────────────┘
        ↓
Response 200: {
  "access_token": "onboarding-token-eyJhbGc...",
  "token_type": "onboarding",
  "next_step": "create_organization"
}
        ↓
┌─────────────────────────────────────────────────┐
│ ETAPA 4: Criar Organization                     │
│ POST /organizations                             │
│ Authorization: Bearer <onboarding-token>        │
└─────────────────────────────────────────────────┘
        ↓
Body: {
  "name": "Empresa ABC"
}
        ↓
┌─────────────────────────────────────────────────┐
│ Sistema cria Organization                       │
│ Sistema cria OrganizationMember (owner)         │
│ Sistema gera JWT FINAL (com organization_id)    │
└─────────────────────────────────────────────────┘
        ↓
Response 201: {
  "access_token": "final-token-eyJhbGc...",
  "refresh_token": "...",
  "organization": {
    "id": "uuid-abc",
    "name": "Empresa ABC",
    "role": "owner"
  }
}
        ↓
┌─────────────────────────────────────────────────┐
│ Frontend redireciona para /dashboard            │
└─────────────────────────────────────────────────┘
```

---

### 3.2 Fluxo Alternativo: Aceitar Convite (Novo Usuário)

```
┌─────────────────────────────────────────────────┐
│ Admin envia convite                             │
│ POST /invites                                   │
└─────────────────────────────────────────────────┘
        ↓
Body: {
  "email": "maria@email.com",
  "role": "member"
}
        ↓
┌─────────────────────────────────────────────────┐
│ Sistema cria Invite (status=pending)            │
│ Sistema envia email com link                    │
└─────────────────────────────────────────────────┘
        ↓
Email: "Clique aqui: /accept-invite?token=xyz"
        ↓
┌─────────────────────────────────────────────────┐
│ Maria clica no link                             │
│ POST /auth/accept-invite                        │
└─────────────────────────────────────────────────┘
        ↓
Body: {
  "token": "invite-token-xyz",
  "password": "Senha123",
  "full_name": "Maria Silva"  // opcional
}
        ↓
┌─────────────────────────────────────────────────┐
│ Sistema cria User (status=ACTIVE)               │
│   - Email já validado (convite valida)          │
│ Sistema cria UserAccount                        │
│ Sistema cria OrganizationMember                 │
│ Sistema marca convite como accepted             │
└─────────────────────────────────────────────────┘
        ↓
Response 200: {
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "organization": {
    "id": "uuid-abc",
    "name": "Empresa ABC",
    "role": "member"
  }
}
        ↓
┌─────────────────────────────────────────────────┐
│ Frontend redireciona para /dashboard            │
│ (Maria já tem organization via convite)         │
└─────────────────────────────────────────────────┘
```

---

## 4. Modelo de Dados

### 4.1 Tabela: users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',  -- inactive, active, suspended
    email_verified_at BIGINT,                        -- Unix timestamp ou null
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status);
```

**Status**:
- `inactive`: Conta criada mas email não verificado (não pode fazer login)
- `active`: Email verificado, conta ativa
- `suspended`: Conta suspensa por admin (não pode fazer login)

---

### 4.2 Tabela: user_accounts

```sql
CREATE TABLE user_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(255),                          -- OPCIONAL (pode ser null)
    avatar_url VARCHAR(500),
    phone VARCHAR(50),
    locale VARCHAR(10) DEFAULT 'pt-BR',
    timezone VARCHAR(50) DEFAULT 'America/Sao_Paulo',
    theme VARCHAR(20) DEFAULT 'light',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

CREATE UNIQUE INDEX idx_user_accounts_user_id ON user_accounts(user_id) WHERE deleted_at IS NULL;
```

**Nota**: `full_name` pode ser preenchido:
- Ao aceitar convite (opcional)
- Editando perfil (`PATCH /me`)
- **Nunca no registro inicial** (POST /auth/register não aceita full_name)

---

### 4.3 Tabela: email_verification_tokens

```sql
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at BIGINT NOT NULL,                      -- 24 horas após criação
    used_at BIGINT,                                  -- quando foi usado (null se não usado)
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_email_verification_tokens_token ON email_verification_tokens(token);
CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
```

**Regras**:
- Token expira em 24 horas
- Após usado, `used_at` é preenchido
- Usuário pode ter múltiplos tokens (reenvio)
- Apenas token mais recente é válido

---

### 4.4 Tabela: invites

```sql
CREATE TABLE invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    invited_by UUID NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',   -- pending, accepted, expired, revoked
    expires_at BIGINT NOT NULL,                      -- 7 dias após criação
    accepted_at BIGINT,
    accepted_by UUID REFERENCES users(id),           -- quem aceitou (pode ser diferente se email genérico)
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT,
    UNIQUE(organization_id, email, status) WHERE status = 'pending'  -- 1 convite pendente por email
);

CREATE INDEX idx_invites_token ON invites(token);
CREATE INDEX idx_invites_email ON invites(email);
CREATE INDEX idx_invites_organization ON invites(organization_id);
```

---

## 5. Validações

### 5.1 Validação de Senha

**Regras de Negócio**:
- **RN-20**: Tamanho entre 8 e 72 caracteres
- **RN-21**: Deve conter pelo menos 1 letra (a-z ou A-Z)
- **RN-22**: Deve conter pelo menos 1 número (0-9)
- **RN-23**: Senha é armazenada usando hash criptográfico seguro
- **RN-24**: Hash deve ser resistente a ataques de força bruta

**Mensagens de Erro (i18n)**:

| Violação                | Código de Erro              | Mensagem (pt-BR)                      |
|-------------------------|-----------------------------|---------------------------------------|
| Tamanho inválido        | `error.password_length`     | "Senha deve ter entre 8 e 72 caracteres" |
| Sem letra               | `error.password_weak`       | "Senha deve conter pelo menos 1 letra" |
| Sem número              | `error.password_weak`       | "Senha deve conter pelo menos 1 número" |
| Hash falhou             | `error.internal_server`     | "Erro ao processar senha"             |

**Exemplos**:

✅ **Senhas Válidas**:
- `Senha123`
- `MyP@ssw0rd`
- `Abc12345`
- `Test1234`

❌ **Senhas Inválidas**:
- `12345678` → Erro: sem letra
- `senhaboa` → Erro: sem número
- `Abc123` → Erro: menos de 8 caracteres
- `a1` → Erro: muito curta

---

### 5.2 Validação de Email

**Regras de Negócio**:
- **RN-25**: Email deve seguir formato RFC 5322 (padrão internacional)
- **RN-26**: Email é normalizado: convertido para lowercase e trimmed
- **RN-27**: Email deve ser único no sistema inteiro (não pode duplicar)
- **RN-28**: Domínios descartáveis são bloqueados (ex: 10minutemail.com)
- **RN-29**: Email deve ter parte local + @ + domínio válido

**Processo de Normalização**:
1. Remove espaços em branco no início/fim
2. Converte para lowercase
3. Exemplo: `  User@Example.COM  ` → `user@example.com`

**Mensagens de Erro (i18n)**:

| Violação                | Código de Erro                        | Mensagem (pt-BR)                   |
|-------------------------|---------------------------------------|------------------------------------|
| Formato inválido        | `error.invalid_email_format`          | "Formato de email inválido"        |
| Email já existe         | `error.email_already_exists`          | "Este email já está cadastrado"    |
| Domínio descartável     | `error.disposable_email_not_allowed`  | "Emails temporários não são permitidos" |

**Exemplos**:

✅ **Emails Válidos**:
- `user@example.com`
- `john.doe@company.co.uk`
- `test+tag@gmail.com`
- `User@Example.COM` → normalizado para `user@example.com`

❌ **Emails Inválidos**:
- `invalid` → Erro: sem @
- `@example.com` → Erro: sem parte local
- `user@` → Erro: sem domínio
- `user@10minutemail.com` → Erro: domínio descartável

---

## 6. Endpoints

### 6.1 POST /auth/register

**Descrição**: Criar conta (sem organization)

**Request**:
```json
{
  "email": "joao@email.com",
  "password": "Senha123"
}
```

**Response 201 Created**:
```json
{
  "user_id": "uuid-123",
  "email": "joao@email.com",
  "status": "inactive",
  "message": "Enviamos um email de verificação para joao@email.com. Verifique sua caixa de entrada."
}
```

**Errors**:
- 400 Bad Request: Validação falhou
- 409 Conflict: Email já existe

---

### 6.2 POST /auth/verify-email

**Descrição**: Verificar email e ativar conta

**Request**:
```json
{
  "token": "verification-token-abc"
}
```

**Response 200 OK**:
```json
{
  "message": "Email verificado com sucesso!",
  "email_verified_at": 1699800000,
  "next_step": "login"
}
```

**Errors**:
- 400 Bad Request: Token inválido
- 410 Gone: Token expirado

---

### 6.3 POST /auth/resend-verification

**Descrição**: Reenviar email de verificação

**Request**:
```json
{
  "email": "joao@email.com"
}
```

**Response 200 OK**:
```json
{
  "message": "Email de verificação reenviado"
}
```

**Errors**:
- 400 Bad Request: Email já verificado
- 404 Not Found: Email não cadastrado

---

### 6.4 POST /auth/login

**Descrição**: Login (retorna token de onboarding se sem organization)

**Request**:
```json
{
  "email": "joao@email.com",
  "password": "Senha123"
}
```

**Response A** - Usuário SEM Organization:
```json
{
  "access_token": "onboarding-token-eyJhbGc...",
  "token_type": "onboarding",
  "expires_in": 3600,
  "next_step": "create_organization",
  "message": "Crie sua organização para começar a usar o AvantPro"
}
```

**Response B** - Usuário COM Organization(s):
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "organization": {
    "id": "uuid-abc",
    "name": "Empresa ABC",
    "role": "owner"
  }
}
```

**Errors**:
- 401 Unauthorized: Credenciais inválidas
- 403 Forbidden: Conta inativa (email não verificado)

---

### 6.5 POST /organizations

**Descrição**: Criar organization (requer token de onboarding)

**Request**:
```http
POST /organizations
Authorization: Bearer <onboarding-token>
Content-Type: application/json

{
  "name": "Empresa ABC"
}
```

**Response 201 Created**:
```json
{
  "access_token": "final-token-eyJhbGc...",
  "refresh_token": "...",
  "organization": {
    "id": "uuid-abc",
    "name": "Empresa ABC",
    "role": "owner",
    "trial_ends_at": 1699900800
  }
}
```

**Errors**:
- 400 Bad Request: Nome inválido
- 401 Unauthorized: Token inválido ou expirado
- 409 Conflict: User já tem organization (não deveria acontecer)

---

### 6.6 POST /auth/accept-invite

**Descrição**: Aceitar convite (cria user ou adiciona a organization)

**Request**:
```json
{
  "token": "invite-token-xyz",
  "password": "Senha123",
  "full_name": "Maria Silva"  // opcional
}
```

**Response 200 OK**:
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "user": {
    "id": "uuid-456",
    "email": "maria@email.com",
    "full_name": "Maria Silva"
  },
  "organization": {
    "id": "uuid-abc",
    "name": "Empresa ABC",
    "role": "member"
  }
}
```

**Errors**:
- 400 Bad Request: Token inválido
- 401 Unauthorized: Senha incorreta (se email existe)
- 409 Conflict: Usuário já é membro
- 410 Gone: Convite expirado

---

### 6.7 POST /invites (Protegido - Admin/Owner)

**Descrição**: Enviar convite

**Request**:
```http
POST /invites
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "email": "maria@email.com",
  "role": "member"
}
```

**Response 201 Created**:
```json
{
  "id": "invite-uuid-xyz",
  "email": "maria@email.com",
  "role": "member",
  "status": "pending",
  "expires_at": 1699900800,
  "invite_url": "https://app.avantpro.com.br/accept-invite?token=xyz"
}
```

---

### 6.8 PATCH /me (Protegido)

**Descrição**: Atualizar perfil do usuário (nome completo, avatar, etc)

**Request**:
```http
PATCH /me
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "full_name": "João Silva Santos",
  "avatar_url": "https://cdn.example.com/avatar.jpg",
  "locale": "en",
  "theme": "dark"
}
```

**Response 200 OK**:
```json
{
  "id": "uuid-123",
  "email": "joao@email.com",
  "full_name": "João Silva Santos",
  "avatar_url": "https://cdn.example.com/avatar.jpg",
  "locale": "en",
  "theme": "dark"
}
```

---

## 7. Regras de Negócio Consolidadas

**Registro**:
- **RN-01**: Email deve ser único no sistema inteiro
- **RN-02**: Conta inicia inativa até verificar email
- **RN-03**: Usuário não pode fazer login com conta inativa

**Verificação de Email**:
- **RN-04**: Token de verificação expira em 24 horas
- **RN-05**: Token é single-use
- **RN-06**: Usuário pode solicitar reenvio ilimitado do email
- **RN-07**: Verificação de email é obrigatória

**Login e Onboarding**:
- **RN-08**: Token de onboarding expira em 1 hora
- **RN-09**: Token de onboarding não permite acessar recursos de negócio
- **RN-10**: Usuário sem organization não pode acessar dashboard

**Criar Organization**:
- **RN-11**: Primeiro membro sempre é owner
- **RN-12**: Organization inicia com trial de 14 dias (configurável)
- **RN-13**: Apenas token de onboarding pode criar organization

**Convites**:
- **RN-14**: Aceitar convite valida email automaticamente
- **RN-15**: Convite expira em 7 dias
- **RN-16**: 1 convite pendente por email por organization
- **RN-17**: Token de convite é single-use

---

## 8. Segurança

### 8.1 Rate Limiting

```
POST /auth/register: 3 tentativas/hora por IP
POST /auth/verify-email: 5 tentativas/hora por IP
POST /auth/resend-verification: 3 tentativas/hora por email
POST /auth/login: 5 tentativas/15min por email
POST /organizations: 3 tentativas/hora por user
POST /auth/accept-invite: 5 tentativas/hora por token
POST /invites: 10 convites/dia por organization
```

### 8.2 Proteção Contra Enumeration Attack

**Problema**: Atacante descobre quais emails estão cadastrados.

**Solução**: Retornar mensagem genérica em `/auth/register`:

```json
// Em vez de: "Email já cadastrado"
// Retornar:
{
  "message": "Se o email não estiver cadastrado, você receberá um email de verificação."
}
```

E enviar email para o endereço existente:
```
Assunto: Tentativa de cadastro detectada

Alguém tentou criar uma conta com este email.
Você já tem uma conta. Clique aqui para fazer login.
```

### 8.3 Proteção Contra Clickjacking em Convites

```http
# Headers de segurança
X-Frame-Options: DENY
Content-Security-Policy: frame-ancestors 'none'
```

Frontend exibe informações claras antes de aceitar:
```
Você está aceitando convite de:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Organization: Empresa ABC
Convidado por: João Silva ✓ Email verificado
Role: Membro
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[ Cancelar ]  [ Confirmar Aceite ]
```

---

## 9. Emails Transacionais

### 9.1 Email de Verificação

**Assunto**: Verifique seu email - AvantPro

**Conteúdo**:
```
Olá,

Por favor, verifique seu email clicando no link abaixo:

[Verificar Email]
https://app.avantpro.com.br/verify-email?token=abc123

Este link expira em 24 horas.

Não solicitou este cadastro? Ignore este email.

━━━━━━━━━━━━━━━━━━━━━━━━
AvantPro - Gestão de Assinaturas
```

---

### 9.2 Email de Convite

**Assunto**: Você foi convidado para [Empresa ABC] no AvantPro

**Conteúdo**:
```
Olá,

João Silva convidou você para se juntar à organização "Empresa ABC" no AvantPro.

Role: Membro
Organização: Empresa ABC

[Aceitar Convite]
https://app.avantpro.com.br/accept-invite?token=xyz

Este convite expira em 7 dias (12/11/2025).

━━━━━━━━━━━━━━━━━━━━━━━━
Não conhece o AvantPro? Saiba mais em https://avantpro.com.br
```

---

### 9.3 Email de Tentativa de Cadastro (Anti-Enumeration)

**Assunto**: Tentativa de cadastro detectada - AvantPro

**Conteúdo**:
```
Olá,

Alguém tentou criar uma conta no AvantPro com este email.

Você já tem uma conta. Clique aqui para fazer login:
https://app.avantpro.com.br/login

Esqueceu sua senha? Clique aqui para redefinir:
https://app.avantpro.com.br/forgot-password

━━━━━━━━━━━━━━━━━━━━━━━━
AvantPro - Gestão de Assinaturas
```

---

## 10. Frontend - Fluxos de UI

### 10.1 Página de Registro (`/signup`)

```
┌────────────────────────────────────┐
│  Crie sua conta no AvantPro        │
├────────────────────────────────────┤
│                                    │
│  Email*                            │
│  [____________________________]    │
│                                    │
│  Senha*                            │
│  [____________________________]    │
│  ⓘ Mínimo 8 caracteres             │
│                                    │
│  [ ] Li e aceito os termos de uso  │
│                                    │
│  [    Criar conta    ]             │
│                                    │
│  Já tem conta? Faça login          │
└────────────────────────────────────┘

Após enviar:
→ Mostrar tela: "Verifique seu email"
```

---

### 10.2 Página de Verificação de Email (`/verify-email`)

```
┌────────────────────────────────────┐
│  ✓ Email verificado!               │
├────────────────────────────────────┤
│                                    │
│  Sua conta foi ativada com sucesso │
│                                    │
│  [    Fazer Login    ]             │
└────────────────────────────────────┘
```

---

### 10.3 Página de Criar Organization (`/onboarding/create-organization`)

```
┌────────────────────────────────────┐
│  Crie sua organização              │
├────────────────────────────────────┤
│                                    │
│  Qual é o nome da sua empresa?     │
│                                    │
│  [____________________________]    │
│                                    │
│  [    Criar Organização    ]       │
└────────────────────────────────────┘

Após criar:
→ Redireciona para /dashboard
```

---

## 11. Testes

### 11.1 Testes Unitários

```go
// User Service
func TestRegister_Success(t *testing.T)
func TestRegister_EmailAlreadyExists(t *testing.T)
func TestRegister_WeakPassword(t *testing.T)
func TestRegister_CreatesInactiveUser(t *testing.T)

// Email Verification
func TestVerifyEmail_Success(t *testing.T)
func TestVerifyEmail_TokenExpired(t *testing.T)
func TestVerifyEmail_TokenAlreadyUsed(t *testing.T)

// Organization Creation
func TestCreateOrganization_Success(t *testing.T)
func TestCreateOrganization_WithoutOnboardingToken(t *testing.T)
func TestCreateOrganization_UserAlreadyHasOrg(t *testing.T)

// Invite Acceptance
func TestAcceptInvite_NewUser(t *testing.T)
func TestAcceptInvite_ExistingUser(t *testing.T)
func TestAcceptInvite_AlreadyMember(t *testing.T)
func TestAcceptInvite_EmailVerifiedAutomatically(t *testing.T)
```

### 11.2 Testes de Integração

```go
func TestCompleteRegistrationFlow(t *testing.T) {
    // 1. Registrar
    response := httpPost("/auth/register", registerPayload)
    assert.Equal(t, 201, response.StatusCode)
    assert.Equal(t, "inactive", response.Status)

    // 2. Verificar no DB
    user := findUserByEmail("joao@email.com")
    assert.Equal(t, "inactive", user.Status)

    // 3. Verificar email
    token := findVerificationToken(user.ID)
    verifyResponse := httpPost("/auth/verify-email", map[string]string{"token": token})
    assert.Equal(t, 200, verifyResponse.StatusCode)

    // 4. Verificar user ativado
    user = findUserByEmail("joao@email.com")
    assert.Equal(t, "active", user.Status)
    assert.NotNil(t, user.EmailVerifiedAt)

    // 5. Login (sem organization)
    loginResponse := httpPost("/auth/login", loginPayload)
    assert.Equal(t, "onboarding", loginResponse.TokenType)

    // 6. Criar organization
    orgResponse := httpPostWithAuth("/organizations", orgPayload, loginResponse.AccessToken)
    assert.Equal(t, 201, orgResponse.StatusCode)
    assert.Equal(t, "owner", orgResponse.Organization.Role)
}
```

---

## 12. Status de Implementação

**Implementado**:
- ❌ Nenhuma funcionalidade ainda

**Pendente (Fase 1 - MVP)**:
- POST /auth/register
- POST /auth/verify-email
- POST /auth/resend-verification
- POST /auth/login (com token de onboarding)
- POST /organizations (com token de onboarding)
- POST /auth/accept-invite
- POST /invites
- PATCH /me
- Email de verificação
- Email de convite
- Rate limiting básico

**Pendente (Fase 2 - Segurança)**:
- Proteção contra enumeration attack
- CAPTCHA adaptativo
- Detecção de emails descartáveis
- GeoIP fraud detection
- Auditoria de tentativas falhadas

---

## 13. Referências

**Specs Relacionadas**:
- `specs/functional/auth.md` - Autenticação, JWT, OAuth2
- `specs/functional/multi-tenancy.md` - Organizations, isolamento
- `specs/technical/security.md` - Implementação JWT, bcrypt
- `specs/technical/validation-i18n.md` - Validação, mensagens

**Padrões Seguidos**:
- Clean Architecture
- Value Objects (Email, Password)
- UnitOfWork (transações atômicas)
- RFC 7807 (Problem Details for HTTP APIs)
