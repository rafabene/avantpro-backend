# Especificações - AvantPro Backend

Este diretório contém todas as especificações do projeto, organizadas por tipo.

## Estrutura

```
specs/
├── functional/         # O QUÊ o sistema faz
│   └── auth.md        # Casos de uso, regras de negócio, fluxos
│
├── technical/         # COMO o sistema funciona
│   ├── architecture.md      # Clean Architecture, camadas, padrões
│   ├── database.md          # Migrations, GORM, PostgreSQL
│   ├── security.md          # JWT, OAuth2, implementação de auth
│   ├── testing.md           # Estratégia de testes, ferramentas
│   └── validation-i18n.md   # Validação e internacionalização
│
└── development/       # FERRAMENTAS para desenvolver
    └── tooling.md     # Air, Docker, Makefile, golangci-lint
```

## Critérios de Classificação

### Functional (Requisitos Funcionais)
- **Foco**: Features, casos de uso, regras de negócio
- **Pergunta**: "O que o sistema deve fazer?"
- **Exemplo**: "Usuário pode fazer login com email/senha"
- **Conteúdo**: Endpoints, fluxos, validações de negócio, roles

### Technical (Requisitos Não-Funcionais)
- **Foco**: Arquitetura, decisões técnicas, implementação
- **Pergunta**: "Como o sistema funciona internamente?"
- **Exemplo**: "Usa JWT com expiração de 15 minutos"
- **Conteúdo**: Código de exemplo, padrões, frameworks, estrutura

### Development (Guias de Desenvolvimento)
- **Foco**: Setup, ferramentas, comandos, workflows
- **Pergunta**: "Como eu desenvolvo/testo/faço deploy?"
- **Exemplo**: "Use `make dev` para iniciar com hot reload"
- **Conteúdo**: Makefile, Docker, CI/CD, convenções

## Como Usar

1. **Entendendo features**: Leia `functional/`
2. **Implementando código**: Consulte `technical/`
3. **Configurando ambiente**: Siga `development/`

## Índice de Documentos

### Functional
- **auth.md** - Autenticação e autorização (login, OAuth2, RBAC)

### Technical
- **architecture.md** - Clean Architecture, separação de camadas
- **database.md** - Migrations com golang-migrate, GORM patterns
- **security.md** - Implementação JWT, OAuth2, middleware auth
- **testing.md** - Unit/Integration/E2E tests, testcontainers
- **validation-i18n.md** - go-playground/validator, mensagens traduzidas

### Development
- **tooling.md** - Air, Docker, Makefile, golangci-lint, govulncheck
