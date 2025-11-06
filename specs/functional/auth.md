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

**Cenário**: Login bem-sucedido com credenciais válidas

```gherkin
Given um usuário cadastrado com email "user@example.com" e senha "Senha123"
When ele envia POST /auth/login com { "email": "user@example.com", "password": "Senha123" }
Then o sistema retorna status 200 OK
And retorna access_token válido por 15 minutos
And retorna refresh_token válido por 7 dias
And o refresh_token é armazenado de forma segura no servidor
And o cliente recebe os tokens no response body
```

**Cenário**: Login com credenciais inválidas

```gherkin
Given um usuário cadastrado com email "user@example.com"
When ele envia POST /auth/login com senha incorreta
Then o sistema retorna status 401 Unauthorized
And retorna error code "invalid_credentials"
And o contador de tentativas falhas é incrementado
And após 5 tentativas falhas, a conta é bloqueada temporariamente
```

**Cenário**: Uso do access token em requisições protegidas

```gherkin
Given um usuário autenticado com access_token válido
When ele faz uma requisição para endpoint protegido com header "Authorization: Bearer <access_token>"
Then o sistema valida o token
And extrai user_id e role do token
And permite acesso ao recurso se autorizado
```

### 3.2 Fluxo OAuth2 (Google)

**Cenário**: Login social com Google - usuário existente

```gherkin
Given um usuário já cadastrado com email "user@gmail.com"
When ele clica em "Login com Google"
Then o sistema redireciona para Google OAuth (GET /auth/oauth/google)
And o usuário autentica no Google e concede permissões
And o Google redireciona de volta com code de autorização
And o sistema troca o code por access_token do Google
And o sistema obtém profile do usuário (email, name, avatar)
And o sistema encontra usuário existente por email
And o sistema atualiza informações do perfil (avatar, nome)
And o sistema gera access_token e refresh_token próprios
Then retorna os tokens JWT para o cliente
```

**Cenário**: Login social com Google - novo usuário

```gherkin
Given um email "newuser@gmail.com" não cadastrado no sistema
When o usuário completa o fluxo OAuth com Google
Then o sistema cria novo usuário com:
  - Email obtido do Google
  - Nome completo obtido do Google
  - Avatar URL obtido do Google
  - Status: active (email já verificado pelo Google)
  - Role: user (padrão)
And o sistema gera access_token e refresh_token
And retorna os tokens para o cliente
```

### 3.3 Fluxo Refresh Token

**Cenário**: Renovar access token expirado

```gherkin
Given um usuário com access_token expirado
And um refresh_token válido (não expirado, válido por 7 dias)
When o cliente envia POST /auth/refresh com { "refresh_token": "..." }
Then o sistema valida o refresh_token
And verifica que o token está armazenado no servidor
And verifica que o token não expirou
And gera novo access_token válido por 15 minutos
Then retorna o novo access_token
```

**Cenário**: Refresh token com rotação habilitada

```gherkin
Given a configuração de rotação de tokens está habilitada
When o cliente usa um refresh_token para obter novo access_token
Then o sistema gera novo access_token
And gera novo refresh_token
And invalida o refresh_token antigo (não pode ser reutilizado)
And retorna ambos os tokens novos
```

**Cenário**: Refresh token inválido ou expirado

```gherkin
Given um refresh_token expirado ou inválido
When o cliente tenta renovar o access_token
Then o sistema retorna status 401 Unauthorized
And retorna error code "invalid_refresh_token"
And o cliente deve fazer login novamente
```

---

## 4. Regras de Negócio

### 4.1 Senhas

- **RN-01**: Tamanho mínimo de 8 caracteres
- **RN-02**: Deve conter pelo menos 1 letra e 1 número
- **RN-03**: Senhas são armazenadas usando hash seguro (nunca em texto plano)
- **RN-04**: Validação aplicada na criação e alteração de senha
- **RN-05**: Após 5 tentativas de login falhas, conta é bloqueada temporariamente

### 4.2 Tokens

- **RN-06**: Access token tem validade de 15 minutos
- **RN-07**: Refresh token tem validade de 7 dias
- **RN-08**: Refresh tokens são armazenados de forma persistente no servidor
- **RN-09**: Logout invalida o refresh token da sessão atual
- **RN-10**: Refresh tokens podem ser rotacionados a cada uso (configurável)
- **RN-11**: Tokens contêm claims: user_id, email, role, permissions
- **RN-12**: Access tokens são stateless (validados apenas por assinatura)

### 4.3 Roles

- **RN-13**: Usuário recebe role "user" por padrão ao criar conta
- **RN-14**: Apenas Admin pode alterar roles de outros usuários
- **RN-15**: Role deve ser um valor válido: admin, user ou guest
- **RN-16**: Usuário não pode remover própria role de admin (previne lockout)

### 4.4 Sessões

- **RN-17**: Usuário pode ter múltiplas sessões ativas simultaneamente (multi-device)
- **RN-18**: Logout padrão invalida apenas a sessão atual (device)
- **RN-19**: Admin pode invalidar todas as sessões de um usuário (revogação total)
- **RN-20**: Usuário pode visualizar lista de sessões ativas (endpoint /auth/sessions)
- **RN-21**: Usuário pode revogar sessões individuais manualmente

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
