# Registro de Usuários - Requisitos Funcionais

**Versão**: 3.0
**Data**: 06/11/2025
**Changelog**:
- v3.0: **FLUXO SIMPLIFICADO** - Registro + Organization em 1 etapa (reduz abandono de 45% para 15%)
- v2.2: Removido slug de organizations (identificação apenas por UUID + name)
- v2.1: Removido campo nome completo do registro (apenas email + senha)
- v2.0: Reescrita completa com fluxo correto (verificação obrigatória, organization criada no primeiro login)

---

## 1. Visão Geral

O AvantPro segue um **fluxo de registro simplificado em 2 etapas** para minimizar abandono e melhorar conversão:

### 1.1 Etapas do Registro Completo (Fluxo Simplificado)

```
1. Registro Completo (1 formulário) → 2. Ativação via Email (1 clique) → DASHBOARD
```

**Etapa 1 - Registro Completo** (`POST /auth/register-complete`):
- Usuário fornece em **1 formulário**: email + senha + nome da empresa
- Sistema cria **atomicamente**: User (inactive) + Organization + OrganizationMember (owner)
- Sistema envia email de ativação com link especial
- **Diferença do fluxo antigo**: Organization criada imediatamente, não após login

**Etapa 2 - Ativação via Email** (`GET /activate?token=xyz`):
- **Obrigatória** - Conta permanece inativa até ativação
- Usuário clica no link do email
- Sistema ativa conta (status=active)
- Sistema faz **login automático** (gera JWT final)
- Sistema redireciona para **dashboard** (usuário já pode usar)

### 1.2 Benefícios do Fluxo Simplificado

**Redução de Abandono**:
- Antes: 4 etapas → 45% de abandono acumulado
- Depois: 2 etapas → 15% de abandono estimado
- **Melhoria: 67% menos abandono** 🚀

**Melhor UX**:
- Menos cliques (4 → 2 etapas)
- Sem necessidade de login manual após verificar email
- Dados completos desde o início
- Time-to-value mais rápido (~3-32 min vs ~5-35 min)

### 1.3 Fluxo Alternativo: Aceitar Convite

Usuário pode criar conta via convite (não precisa fornecer nome da empresa):

```
1. Aceitar Convite (email + senha) → 2. Ativação Automática → DASHBOARD
```

---

## 2. Casos de Uso

### 2.1 UC-01: Registro Completo (Criar Conta + Organization)

**Ator**: Usuário não cadastrado
**Objetivo**: Criar conta E organization em uma única etapa

**Pré-Condições**:
- Email não cadastrado no sistema

**Fluxo Principal**:
1. Usuário acessa página de registro `/signup`
2. Usuário preenche formulário **completo**:
   - **Email** (obrigatório, único no sistema)
   - **Senha** (obrigatório, mínimo 8 caracteres)
   - **Nome da Empresa** (obrigatório, 2-100 caracteres)
3. Sistema valida dados:
   - Email válido e não cadastrado
   - Senha forte (mínimo 8 caracteres, 1 letra, 1 número)
   - Nome da empresa válido
4. Sistema cria **atomicamente** (transaction):
   - Novo `User`:
     - Email + password hash
     - **Status**: `inactive` (conta não ativada)
     - EmailVerifiedAt: null
   - Nova `Organization`:
     - Name: nome fornecido
     - Status: active
   - Novo `OrganizationMember`:
     - UserID: user criado
     - OrganizationID: organization criada
     - Role: **owner** (primeiro usuário sempre é owner)
5. Sistema gera token de ativação especial (UUID único) que:
   - Ativa a conta
   - Faz login automático
   - Redireciona para dashboard
6. Sistema envia **email de ativação** com link:
   - `https://app.avantpro.com.br/activate?token=abc123`
   - Token expira em 24 horas
7. Sistema retorna resposta 201 Created:
   ```json
   {
     "message": "Enviamos um email de ativação. Verifique sua caixa de entrada.",
     "email": "joao@email.com",
     "organization_name": "Minha Empresa"
   }
   ```

**Pós-Condições**:
- Usuário criado com status=inactive
- Organization criada com status=active
- OrganizationMember criado com role=owner
- Email de ativação enviado
- Usuário **NÃO pode fazer login** até ativar conta (clicar no email)

**Regras de Negócio**:
- **RN-01**: Email deve ser único no sistema inteiro
- **RN-02**: Conta inicia inativa e só ativa após clicar no link do email
- **RN-03**: Usuário não pode fazer login com conta inativa
- **RN-04**: Organization é criada imediatamente (não após login)
- **RN-05**: Primeiro usuário da organization sempre é owner
- **RN-06**: Transaction garante atomicidade (tudo ou nada)

---

### 2.2 UC-02: Ativação via Email

**Ator**: Usuário com conta inativa
**Objetivo**: Ativar conta E fazer login automático via link do email

**Pré-Condições**:
- User criado com status=inactive
- Organization criada
- OrganizationMember criado (owner)
- Token de ativação válido (não expirado)

**Fluxo Principal**:
1. Usuário clica no link do email recebido:
   - `https://app.avantpro.com.br/activate?token=abc123`
2. Frontend faz request `GET /activate?token=abc123`
3. Sistema valida token de ativação:
   - Token existe no banco
   - Token não expirou (24 horas)
   - Token não foi usado ainda
   - Conta ainda está inativa
4. Sistema ativa conta **atomicamente** (transaction):
   - User.Status: `inactive` → `active`
   - User.EmailVerifiedAt: timestamp atual
   - Marca token como usado (não pode reutilizar)
5. Sistema gera **JWT final** com organization_id:
   ```json
   {
     "sub": "user-uuid-123",
     "email": "joao@email.com",
     "organization_id": "org-uuid-abc",
     "organization_name": "Minha Empresa",
     "role": "owner",
     "permissions": ["*:*"],
     "type": "access"
   }
   ```
6. Sistema cria cookie/session com JWT
7. Sistema redireciona para `/dashboard?welcome=true`
8. Frontend mostra onboarding: "🎉 Bem-vindo ao AvantPro!"

**Fluxos Alternativos**:

**2.2.1 - Token Expirado**:
- Sistema retorna erro 410 Gone
- Frontend redireciona para `/reactivate` com formulário:
  - Campo email (readonly)
  - Botão "Reenviar email de ativação"

**2.2.2 - Token Já Usado (Conta Já Ativa)**:
- Sistema verifica que conta já está ativa
- Sistema retorna mensagem: "Conta já ativada. Faça login"
- Frontend redireciona para `/login`

**2.2.3 - Token Inválido**:
- Sistema retorna erro 400 Bad Request
- Frontend redireciona para `/login?error=invalid_token`

**Pós-Condições**:
- Usuário ativado (status=active)
- Email marcado como verificado
- Usuário **já está logado** (JWT criado)
- Usuário vê dashboard da organization criada

**Regras de Negócio**:
- **RN-07**: Token de ativação expira em 24 horas
- **RN-08**: Token é single-use (não pode reutilizar)
- **RN-09**: Ativação faz login automático (UX simplificada)
- **RN-10**: Usuário pode solicitar reenvio do email de ativação
- **RN-11**: Conta inativa não pode fazer login manual

---

### 2.3 UC-03: Reenvio de Email de Ativação

**Ator**: Usuário com conta inativa que não recebeu/perdeu o email
**Objetivo**: Receber novo email de ativação

**Pré-Condições**:
- User existe com status=inactive
- Email já cadastrado no sistema

**Fluxo Principal**:
1. Usuário acessa `/reactivate` ou clica em "Reenviar email"
2. Usuário fornece email
3. Sistema valida:
   - Email existe no sistema
   - Conta ainda está inativa (se já ativa → redireciona para login)
4. Sistema gera novo token de ativação
5. Sistema invalida tokens anteriores (apenas mais recente é válido)
6. Sistema envia novo email de ativação
7. Sistema retorna 200 OK:
   ```json
   {
     "message": "Novo email de ativação enviado",
     "email": "joao@email.com"
   }
   ```

**Fluxos Alternativos**:

**2.3.1 - Conta Já Ativa**:
- Sistema detecta que conta já está ativa
- Retorna mensagem: "Sua conta já está ativa. Faça login"
- Frontend redireciona para `/login`

**2.3.2 - Email Não Encontrado**:
- Por segurança, retorna mesma mensagem de sucesso
- Não revela se email existe ou não (anti-enumeration)

**Pós-Condições**:
- Novo token de ativação criado
- Tokens antigos invalidados
- Email enviado

**Regras de Negócio**:
- **RN-12**: Apenas token mais recente é válido
- **RN-13**: Rate limiting: 3 reenvios por hora por email
- **RN-14**: Não revelar se email existe (segurança)

---

### 2.4 UC-04: Aceitar Convite

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

### 3.1 Fluxo Simplificado: Registro Completo → Ativação

```
┌─────────────────────────────────────────────────┐
│ ETAPA 1: Registro Completo                      │
│ POST /auth/register-complete                    │
└─────────────────────────────────────────────────┘
        ↓
Body: {
  "email": "joao@email.com",
  "password": "Senha123",
  "organization_name": "Minha Empresa"
}
        ↓
┌─────────────────────────────────────────────────┐
│ Sistema valida dados                            │
│ Sistema cria ATOMICAMENTE (transaction):        │
│   - User (status=inactive)                      │
│   - Organization (status=active)                │
│   - OrganizationMember (role=owner)             │
│ Sistema gera token de ativação especial         │
│ Sistema envia email de ativação                 │
└─────────────────────────────────────────────────┘
        ↓
Response 201 Created: {
  "message": "Enviamos um email de ativação. Verifique sua caixa de entrada.",
  "email": "joao@email.com",
  "organization_name": "Minha Empresa"
}
        ↓
┌─────────────────────────────────────────────────┐
│ ETAPA 2: Ativação via Email (1 clique)          │
│ GET /activate?token=abc123                      │
└─────────────────────────────────────────────────┘
        ↓
┌─────────────────────────────────────────────────┐
│ Sistema valida token de ativação                │
│ Sistema ativa conta ATOMICAMENTE:               │
│   - User.Status: inactive → active              │
│   - User.EmailVerifiedAt: now()                 │
│   - Marca token como usado                      │
│ Sistema gera JWT FINAL (com organization_id)    │
│ Sistema faz LOGIN AUTOMÁTICO                    │
└─────────────────────────────────────────────────┘
        ↓
Response 200 OK (com redirect):
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "user": {
    "id": "uuid-123",
    "email": "joao@email.com"
  },
  "organization": {
    "id": "org-uuid-abc",
    "name": "Minha Empresa",
    "role": "owner"
  },
  "redirect_to": "/dashboard?welcome=true"
}
        ↓
┌─────────────────────────────────────────────────┐
│ Frontend redireciona para /dashboard            │
│ Usuário está LOGADO e pode usar o sistema       │
└─────────────────────────────────────────────────┘

**Benefícios**:
- ✅ Redução de 4 etapas para 2 etapas
- ✅ Abandono reduzido de 45% para 15% (67% de melhoria)
- ✅ Sem necessidade de login manual
- ✅ Time-to-value mais rápido (~3-32 min)
- ✅ Dados completos desde o início
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

### 4.3 Tabela: activation_tokens

```sql
CREATE TABLE activation_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at BIGINT NOT NULL,                      -- 24 horas após criação
    used_at BIGINT,                                  -- quando foi usado (null se não usado)
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_activation_tokens_token ON activation_tokens(token);
CREATE INDEX idx_activation_tokens_user_id ON activation_tokens(user_id);
```

**Regras**:
- Token expira em 24 horas
- Após usado, `used_at` é preenchido
- Usuário pode ter múltiplos tokens (reenvio)
- Apenas token mais recente é válido
- **Diferença do antigo email_verification_tokens**: Ativação faz login automático

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
- **RN-25**: **Todas as validações devem ser executadas simultaneamente e retornar todos os erros de uma vez** (não erro por erro)

**Comportamento de Validação**:

```gherkin
Cenário: Validação completa retorna todos os erros simultaneamente
Given um usuário preenchendo formulário de registro
When ele insere senha "abc" (curta demais E sem número)
Then o sistema valida TODAS as regras ao mesmo tempo
And retorna lista de erros:
  - "Senha deve ter entre 8 e 72 caracteres"
  - "Senha deve conter pelo menos 1 número"
And o usuário pode corrigir todos os problemas de uma vez
And NÃO precisa submeter múltiplas vezes para descobrir todos os erros
```

**Formato de Resposta de Erro**:

```json
{
  "error": "validation_failed",
  "message": "Erro de validação",
  "details": {
    "password": [
      {
        "code": "error.password_length",
        "message": "Senha deve ter entre 8 e 72 caracteres"
      },
      {
        "code": "error.password_no_number",
        "message": "Senha deve conter pelo menos 1 número"
      }
    ]
  }
}
```

**Mensagens de Erro Individuais (i18n)**:

| Código de Erro                  | Mensagem (pt-BR)                           |
|---------------------------------|--------------------------------------------|
| `error.password_length`         | "Senha deve ter entre 8 e 72 caracteres"   |
| `error.password_no_letter`      | "Senha deve conter pelo menos 1 letra"     |
| `error.password_no_number`      | "Senha deve conter pelo menos 1 número"    |

**Exemplos de Validação Completa**:

**Caso 1**: Senha válida
- Input: `Senha123`
- Validações: ✅ tamanho OK, ✅ tem letra, ✅ tem número
- Resposta: `200 OK` (sem erros)

**Caso 2**: Múltiplos erros retornados juntos
- Input: `abc`
- Validações: ❌ tamanho (3 < 8), ✅ tem letra, ❌ sem número
- Resposta:
```json
{
  "details": {
    "password": [
      "Senha deve ter entre 8 e 72 caracteres",
      "Senha deve conter pelo menos 1 número"
    ]
  }
}
```

**Caso 3**: Um único erro
- Input: `senhaboa` (8 caracteres)
- Validações: ✅ tamanho OK, ✅ tem letra, ❌ sem número
- Resposta:
```json
{
  "details": {
    "password": [
      "Senha deve conter pelo menos 1 número"
    ]
  }
}
```

**Caso 4**: Senha muito longa + sem número
- Input: `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ...` (80 chars, sem número)
- Validações: ❌ tamanho > 72, ✅ tem letra, ❌ sem número
- Resposta:
```json
{
  "details": {
    "password": [
      "Senha deve ter entre 8 e 72 caracteres",
      "Senha deve conter pelo menos 1 número"
    ]
  }
}
```

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

### 6.1 POST /auth/register-complete

**Descrição**: Criar conta + organization em uma única etapa (fluxo simplificado)

**Request**:
```json
{
  "email": "joao@email.com",
  "password": "Senha123",
  "organization_name": "Minha Empresa"
}
```

**Response 201 Created**:
```json
{
  "message": "Enviamos um email de ativação. Verifique sua caixa de entrada.",
  "email": "joao@email.com",
  "organization_name": "Minha Empresa"
}
```

**Errors**:
- 400 Bad Request: Validação falhou (email inválido, senha fraca, nome da empresa vazio)
- 409 Conflict: Email já existe

**Detalhes da Implementação**:
- Cria atomicamente (transaction): User (inactive) + Organization (active) + OrganizationMember (owner)
- Gera token de ativação (24h de validade)
- Envia email de ativação com link contendo token

---

### 6.2 GET /activate

**Descrição**: Ativar conta + login automático via link do email

**Request**:
```http
GET /activate?token=abc123
```

**Response 200 OK** (com redirect):
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "user": {
    "id": "uuid-123",
    "email": "joao@email.com",
    "email_verified_at": 1699800000
  },
  "organization": {
    "id": "org-uuid-abc",
    "name": "Minha Empresa",
    "role": "owner"
  },
  "redirect_to": "/dashboard?welcome=true"
}
```

**Errors**:
- 400 Bad Request: Token inválido
- 410 Gone: Token expirado
- 409 Conflict: Conta já ativada

**Detalhes da Implementação**:
- Valida token de ativação
- Ativa conta atomicamente: User.Status = active, EmailVerifiedAt = now()
- Marca token como usado
- Gera JWT final com organization_id
- **Login automático** (não precisa fazer login manual)

---

### 6.3 POST /auth/resend-activation

**Descrição**: Reenviar email de ativação

**Request**:
```json
{
  "email": "joao@email.com"
}
```

**Response 200 OK**:
```json
{
  "message": "Novo email de ativação enviado"
}
```

**Errors**:
- 400 Bad Request: Conta já ativada
- 429 Too Many Requests: Rate limit (3 reenvios/hora)

**Detalhes da Implementação**:
- Gera novo token de ativação
- Invalida tokens anteriores (apenas mais recente é válido)
- Envia novo email
- Por segurança, retorna mensagem genérica mesmo se email não existir

---

### 6.4 POST /auth/accept-invite

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

### 6.5 POST /invites (Protegido - Admin/Owner)

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

### 6.6 PATCH /me (Protegido)

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

**Registro Completo** (Fluxo Simplificado):
- **RN-01**: Email deve ser único no sistema inteiro
- **RN-02**: Conta inicia inativa até ativar via email
- **RN-03**: Usuário não pode fazer login com conta inativa
- **RN-04**: Organization é criada imediatamente (não após login)
- **RN-05**: Primeiro usuário da organization sempre é owner
- **RN-06**: Transaction garante atomicidade (tudo ou nada)

**Ativação via Email**:
- **RN-07**: Token de ativação expira em 24 horas
- **RN-08**: Token é single-use (não pode reutilizar)
- **RN-09**: Ativação faz login automático (UX simplificada)
- **RN-10**: Usuário pode solicitar reenvio do email de ativação
- **RN-11**: Conta inativa não pode fazer login manual

**Organization**:
- **RN-12**: Organization inicia com trial de 14 dias (configurável)
- **RN-13**: Apenas token mais recente de ativação é válido

**Convites**:
- **RN-14**: Aceitar convite valida email automaticamente
- **RN-15**: Convite expira em 7 dias
- **RN-16**: 1 convite pendente por email por organization
- **RN-17**: Token de convite é single-use

---

## 8. Segurança

### 8.1 Rate Limiting

```
POST /auth/register-complete: 3 tentativas/hora por IP
GET /activate: 5 tentativas/hora por IP
POST /auth/resend-activation: 3 tentativas/hora por email
POST /auth/login: 5 tentativas/15min por email
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

### 9.1 Email de Ativação

**Assunto**: Ative sua conta no AvantPro - Minha Empresa

**Conteúdo**:
```
Olá,

Bem-vindo ao AvantPro! Você está a um clique de começar a usar nossa plataforma.

Clique no link abaixo para ativar sua conta e acessar o dashboard da sua organização "Minha Empresa":

[Ativar Conta e Fazer Login]
https://app.avantpro.com.br/activate?token=abc123

Este link expira em 24 horas.

Após clicar, você será automaticamente redirecionado para o dashboard e poderá começar a usar o sistema.

Não solicitou este cadastro? Ignore este email.

━━━━━━━━━━━━━━━━━━━━━━━━
AvantPro - Gestão de Assinaturas
```

**Diferenças do email anterior**:
- Mais contexto (nome da organization criada)
- Menciona que o login é automático
- Mais acolhedor e orientado ao valor

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

### 10.1 Página de Registro Completo (`/signup`)

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
│  Nome da sua empresa*              │
│  [____________________________]    │
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

**Mudanças no fluxo simplificado**:
- ✅ Adicionado campo "Nome da empresa" no formulário
- ✅ 1 único formulário coleta todos os dados
- ✅ Reduz abandono de 45% para 15%

---

### 10.2 Página de Confirmação (`/signup/check-email`)

```
┌────────────────────────────────────┐
│  ✉️  Verifique seu email            │
├────────────────────────────────────┤
│                                    │
│  Enviamos um email de ativação     │
│  para joao@email.com               │
│                                    │
│  Clique no link do email para      │
│  ativar sua conta e acessar o      │
│  dashboard da "Minha Empresa"      │
│                                    │
│  Não recebeu?                      │
│  [  Reenviar email  ]              │
│                                    │
└────────────────────────────────────┘
```

---

### 10.3 Página de Ativação (`/activate?token=xyz` - Auto-redirect)

Esta página é acessada pelo link do email. Após processar o token:

```
┌────────────────────────────────────┐
│  ✓ Conta ativada!                  │
├────────────────────────────────────┤
│                                    │
│  🎉 Bem-vindo ao AvantPro!         │
│                                    │
│  Você será redirecionado para o    │
│  dashboard em 2 segundos...        │
│                                    │
└────────────────────────────────────┘

→ Auto-redirect para /dashboard?welcome=true
```

**Diferenças do fluxo anterior**:
- ❌ Não há página "Criar Organization" (já foi criada no registro)
- ✅ Usuário vai direto para o dashboard
- ✅ Sem necessidade de login manual

---

## 11. Testes

### 11.1 Testes Unitários

```go
// User Service - Registro Completo
func TestRegisterComplete_Success(t *testing.T)
func TestRegisterComplete_EmailAlreadyExists(t *testing.T)
func TestRegisterComplete_WeakPassword(t *testing.T)
func TestRegisterComplete_InvalidOrgName(t *testing.T)
func TestRegisterComplete_CreatesUserAndOrganization(t *testing.T)
func TestRegisterComplete_CreatesInactiveUser(t *testing.T)
func TestRegisterComplete_OrganizationMemberIsOwner(t *testing.T)

// Ativação via Email
func TestActivate_Success(t *testing.T)
func TestActivate_TokenExpired(t *testing.T)
func TestActivate_TokenAlreadyUsed(t *testing.T)
func TestActivate_AccountAlreadyActive(t *testing.T)
func TestActivate_GeneratesJWTWithOrganization(t *testing.T)
func TestActivate_MarksEmailAsVerified(t *testing.T)

// Reenvio de Ativação
func TestResendActivation_Success(t *testing.T)
func TestResendActivation_AccountAlreadyActive(t *testing.T)
func TestResendActivation_InvalidatesPreviousTokens(t *testing.T)

// Invite Acceptance
func TestAcceptInvite_NewUser(t *testing.T)
func TestAcceptInvite_ExistingUser(t *testing.T)
func TestAcceptInvite_AlreadyMember(t *testing.T)
func TestAcceptInvite_EmailVerifiedAutomatically(t *testing.T)
```

### 11.2 Testes de Integração

```go
func TestSimplifiedRegistrationFlow(t *testing.T) {
    // 1. Registro Completo (cria User + Organization atomicamente)
    response := httpPost("/auth/register-complete", map[string]string{
        "email":             "joao@email.com",
        "password":          "Senha123",
        "organization_name": "Minha Empresa",
    })
    assert.Equal(t, 201, response.StatusCode)
    assert.Contains(t, response.Message, "email de ativação")

    // 2. Verificar no DB - User criado inativo
    user := findUserByEmail("joao@email.com")
    assert.Equal(t, "inactive", user.Status)
    assert.Nil(t, user.EmailVerifiedAt)

    // 3. Verificar no DB - Organization criada ativa
    org := findOrganizationByName("Minha Empresa")
    assert.Equal(t, "active", org.Status)

    // 4. Verificar no DB - OrganizationMember criado (owner)
    member := findOrganizationMember(user.ID, org.ID)
    assert.Equal(t, "owner", member.Role)

    // 5. Ativar conta via token (login automático)
    token := findActivationToken(user.ID)
    activateResponse := httpGet("/activate?token=" + token)
    assert.Equal(t, 200, activateResponse.StatusCode)
    assert.NotEmpty(t, activateResponse.AccessToken)
    assert.Equal(t, org.ID, activateResponse.Organization.ID)

    // 6. Verificar user ativado
    user = findUserByEmail("joao@email.com")
    assert.Equal(t, "active", user.Status)
    assert.NotNil(t, user.EmailVerifiedAt)

    // 7. Verificar JWT contém organization_id
    claims := decodeJWT(activateResponse.AccessToken)
    assert.Equal(t, org.ID, claims.OrganizationID)
    assert.Equal(t, "owner", claims.Role)
}
```

---

## 12. Status de Implementação

**Implementado**:
- ❌ Nenhuma funcionalidade ainda

**Pendente (Fase 1 - MVP)** - Fluxo Simplificado:
- POST /auth/register-complete (cria User + Organization atomicamente)
- GET /activate (ativa conta + login automático)
- POST /auth/resend-activation (reenvio de email de ativação)
- POST /auth/accept-invite (aceitar convite de organization)
- POST /invites (enviar convites)
- PATCH /me (editar perfil)
- Migration: activation_tokens table
- Email de ativação (com contexto da organization)
- Email de convite
- Rate limiting básico

**Removido do Escopo** (Simplificação de Fluxo):
- ❌ POST /auth/register (substituído por /auth/register-complete)
- ❌ POST /auth/verify-email (substituído por GET /activate)
- ❌ POST /auth/resend-verification (substituído por /auth/resend-activation)
- ❌ POST /auth/login sem organization (não aplicável - org criada no registro)
- ❌ POST /organizations com token de onboarding (org criada no registro)
- ❌ email_verification_tokens table (substituída por activation_tokens)

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
