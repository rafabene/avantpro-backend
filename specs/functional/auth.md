# Autenticação e Autorização - Requisitos Funcionais

**Versão**: 2.0
**Data**: 05/11/2025

---

## 1. Casos de Uso

### 1.1 Autenticação

**UC-01: Login com Email/Password**
- Usuário fornece email e senha
- Sistema valida credenciais
- Sistema retorna access token (15min) e refresh token (7 dias)
- Tokens são usados para acessar recursos protegidos

**UC-02: Registro de Usuário**
- Usuário fornece email, nome e senha
- Sistema valida:
  - Email único no sistema
  - Senha forte (mínimo 8 caracteres)
  - Nome válido
- Sistema cria conta e retorna tokens

**UC-03: Refresh Token**
- Cliente envia refresh token expirado/próximo de expirar
- Sistema valida refresh token
- Sistema gera novo access token
- Opcionalmente rotaciona refresh token

**UC-04: Logout**
- Cliente envia request de logout
- Sistema invalida refresh token atual
- Access token expira naturalmente (stateless)

**UC-05: Login Social (OAuth2)**
- Usuário clica em "Login com Google/GitHub"
- Sistema redireciona para provider OAuth
- Provider autentica usuário
- Sistema recebe callback com código
- Sistema troca código por tokens
- Sistema cria/atualiza usuário no banco
- Sistema retorna JWT tokens próprios

---

## 2. RBAC (Role-Based Access Control)

### 2.1 Roles

**Admin**
- Acesso total ao sistema
- Pode gerenciar todos os usuários
- Pode acessar endpoints administrativos
- Pode modificar permissões

**User** (padrão)
- Acesso aos próprios recursos
- Pode criar/editar suas assinaturas
- Pode visualizar próprio perfil
- Não pode acessar recursos de outros usuários

**Guest**
- Acesso apenas leitura
- Pode visualizar conteúdo público
- Não pode criar/editar recursos

### 2.2 Permissions

Formato: `resource.action`

**Users**
- `users.read` - Visualizar usuários
- `users.write` - Criar/editar usuários
- `users.delete` - Deletar usuários

**Subscriptions**
- `subscriptions.read` - Visualizar assinaturas
- `subscriptions.write` - Criar/editar assinaturas
- `subscriptions.cancel` - Cancelar assinaturas

**Payments**
- `payments.read` - Visualizar pagamentos
- `payments.process` - Processar pagamentos

### 2.3 Mapeamento Role → Permissions

```
Admin:
  - users.*
  - subscriptions.*
  - payments.*

User:
  - users.read (apenas próprio)
  - subscriptions.read (apenas próprias)
  - subscriptions.write (apenas próprias)
  - subscriptions.cancel (apenas próprias)
  - payments.read (apenas próprios)

Guest:
  - (nenhuma permissão de escrita)
```

---

## 3. Fluxos de Autenticação

### 3.1 Fluxo Login Email/Password

```
1. POST /auth/login
   Body: { email, password }

2. Sistema valida credenciais
   - Busca usuário por email
   - Compara senha (bcrypt)

3. Se válido:
   - Gera access_token (15min)
   - Gera refresh_token (7 dias)
   - Armazena refresh_token no Redis
   - Retorna tokens

4. Cliente armazena tokens
   - access_token: memória (não localStorage)
   - refresh_token: httpOnly cookie (seguro)

5. Cliente usa access_token em requisições
   Header: Authorization: Bearer <access_token>
```

### 3.2 Fluxo OAuth2 (Google)

```
1. GET /auth/oauth/google
   - Sistema redireciona para Google OAuth

2. Usuário autentica no Google
   - Concede permissões

3. Google redireciona para callback
   GET /auth/oauth/callback?code=xyz&state=abc

4. Sistema troca code por tokens Google
   - Obtém access_token do Google
   - Obtém profile info (email, name, avatar)

5. Sistema busca/cria usuário
   - Se email existe: atualiza info
   - Se não existe: cria novo usuário

6. Sistema gera JWT próprio
   - access_token (15min)
   - refresh_token (7 dias)

7. Retorna tokens para cliente
```

### 3.3 Fluxo Refresh Token

```
1. Access token expira (15min)

2. Cliente detecta 401 Unauthorized

3. POST /auth/refresh
   Body: { refresh_token }

4. Sistema valida refresh_token
   - Verifica assinatura JWT
   - Verifica se existe no Redis
   - Verifica se não expirou (7 dias)

5. Se válido:
   - Gera novo access_token (15min)
   - Opcionalmente gera novo refresh_token (rotação)
   - Invalida refresh_token antigo se rotação

6. Retorna novo access_token
```

---

## 4. Regras de Negócio

### 4.1 Senhas

- **Tamanho mínimo**: 8 caracteres
- **Requisitos**: Pelo menos 1 letra e 1 número
- **Hash**: bcrypt (cost 12)
- **Validação**: Na criação e alteração de senha

### 4.2 Tokens

- **Access Token**: 15 minutos de validade
- **Refresh Token**: 7 dias de validade
- **Armazenamento**: Refresh tokens no Redis
- **Revogação**: Logout invalida refresh token
- **Rotação**: Refresh token pode ser rotacionado a cada uso

### 4.3 Roles

- **Padrão**: User (ao criar conta)
- **Alteração**: Apenas Admin pode alterar roles
- **Validação**: Role deve ser um valor válido (admin, user, guest)

### 4.4 Sessões

- **Multi-device**: Usuário pode ter múltiplas sessões ativas
- **Logout device**: Logout invalida apenas sessão atual
- **Logout all**: Admin pode invalidar todas as sessões de um usuário

---

## 5. Endpoints

### 5.1 Públicos (sem autenticação)

```
POST   /auth/register       - Registrar novo usuário
POST   /auth/login          - Login com email/senha
POST   /auth/refresh        - Renovar access token
GET    /auth/oauth/google   - Iniciar OAuth Google
GET    /auth/oauth/github   - Iniciar OAuth GitHub
GET    /auth/oauth/callback - Callback OAuth
```

### 5.2 Protegidos (requer autenticação)

```
POST   /auth/logout         - Logout (invalida refresh token)
GET    /auth/me             - Obter usuário atual
POST   /auth/password       - Alterar senha
```

### 5.3 Admin apenas

```
POST   /admin/users/:id/role           - Alterar role de usuário
POST   /admin/users/:id/revoke-tokens  - Invalidar todos os tokens
GET    /admin/users                    - Listar todos os usuários
```

---

## 6. Casos de Erro

### 6.1 Login Falhou

**Cenário**: Email/senha incorretos
**Response**: 401 Unauthorized
```json
{
  "error": "invalid_credentials",
  "message": "Email ou senha incorretos"
}
```

### 6.2 Token Expirado

**Cenário**: Access token expirado
**Response**: 401 Unauthorized
```json
{
  "error": "token_expired",
  "message": "Token expirado, use refresh token"
}
```

### 6.3 Token Inválido

**Cenário**: Token malformado ou assinatura inválida
**Response**: 401 Unauthorized
```json
{
  "error": "invalid_token",
  "message": "Token inválido"
}
```

### 6.4 Permissão Negada

**Cenário**: Usuário sem permissão para recurso
**Response**: 403 Forbidden
```json
{
  "error": "forbidden",
  "message": "Você não tem permissão para acessar este recurso"
}
```

### 6.5 Email Já Existe

**Cenário**: Tentativa de registro com email existente
**Response**: 409 Conflict
```json
{
  "error": "email_already_exists",
  "message": "Este email já está cadastrado"
}
```

---

## 7. Status de Implementação

**Implementado**:
- ✅ Estrutura de roles (entities.Role)
- ✅ Permissions mapping
- ✅ RBAC na camada de domínio (User.HasPermission)

**Pendente**:
- ⏳ JWT generation/validation
- ⏳ Auth service (login, register, refresh)
- ⏳ Auth handlers (/auth/login, /auth/register)
- ⏳ OAuth2 integration (Google, GitHub)
- ⏳ Middleware de autenticação
- ⏳ Middleware de autorização (RBAC)
- ⏳ Refresh token storage (Redis)
- ⏳ Password hashing (bcrypt)
