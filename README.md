
![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Makefile](https://img.shields.io/badge/Makefile-429800?style=for-the-badge&logo=gnu-make&logoColor=white)
![Swagger](https://img.shields.io/badge/Swagger-85EA2D?style=for-the-badge&logo=swagger&logoColor=black)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)
![Neon](https://img.shields.io/badge/Neon-00E599?style=for-the-badge&logo=neon&logoColor=black)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-FF6600?style=for-the-badge&logo=rabbitmq&logoColor=white)
![Gin](https://img.shields.io/badge/Gin-009485?style=for-the-badge&logo=gin&logoColor=white)
![GORM](https://img.shields.io/badge/GORM-AF1921?style=for-the-badge)
![Render](https://img.shields.io/badge/Render-46E3B7?style=for-the-badge&logo=render&logoColor=white)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-FF6600?style=for-the-badge&logo=rabbitmq&logoColor=white)
![Docker Compose](https://img.shields.io/badge/Docker%20Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![JWT](https://img.shields.io/badge/JWT-000000?style=for-the-badge&logo=jsonwebtokens&logoColor=white)
![Zap](https://img.shields.io/badge/Zap%20Logger-7B00FF?style=for-the-badge)
![Validator v10](https://img.shields.io/badge/Validator%20v10-4CAF50?style=for-the-badge)

![Status](https://img.shields.io/badge/Status-Ativo-brightgreen?style=for-the-badge)


# Products CRUD API com RabbitMQ, JWT e Notificações por Email


> API de CRUD de produtos com autenticação via JWT, filas RabbitMQ e envio de notificações por email a cada operação realizada.
> Segue **Arquitetura Limpa (Clean Architecture)**, separando camadas de domínio, caso de uso, infraestrutura e interface.

---

### 🌐 URL de Produção

A API está disponível em produção no Render: [https://products-crud-kh.onrender.com](https://products-crud-kh.onrender.com)  

> A URL principal não retorna conteúdo, serve apenas para expor as rotas.  

Swagger disponível em: [https://products-crud-kh.onrender.com/swagger/index.html](https://products-crud-kh.onrender.com/swagger/index.html)

---

### 📖 Descrição do Projeto

Este projeto foi desenvolvido em **Golang** com arquitetura modular e organizada em **camadas**, seguindo os princípios da **Arquitetura Limpa**:

- **Domain**: Modelos e regras de negócio (produto, usuário).  
- **Usecase**: Casos de uso, lógica de aplicação.  
- **Handler / API**: Camada de interface, endpoints HTTP com Gin.  
- **Infrastructure**: Banco de dados (Neon/PostgreSQL), RabbitMQ, envio de emails, logging.  

### Funcionalidades

#### CRUD de Produtos
- Criar, ler, atualizar e deletar produtos.
- Validação automática de dados.

#### Autenticação JWT
- Geração de token JWT ao logar.
- Proteção de rotas sensíveis.

#### RabbitMQ
- Publica eventos em filas (`publisher.go`).
- Consome eventos (`consumer.go`) e envia emails de notificação.

#### Email Notifications
- Emails detalhados de operações CRUD.
- Configuração flexível via `.env`.

#### Swagger
- Documentação interativa disponível em ambiente de desenvolvimento e em produção.

---

#### Estrutura do Projeto

```text
.
├── cmd
│   ├── api
│   │   ├── docs
│   │   │   ├── docs.go
│   │   │   ├── swagger.json
│   │   │   └── swagger.yaml
│   │   └── main.go
│   └── consumer
│       └── consumer.go
├── internal
│   ├── config
│   │   └── config.go
│   ├── domain
│   │   ├── messaging
│   │   │   ├── consumer.go
│   │   │   └── publisher.go
│   │   ├── model
│   │   │   ├── product.go
│   │   │   └── user.go
│   │   ├── repository
│   │   │   ├── product.go
│   │   │   └── repository.go
│   │   └── usecase
│   │       ├── usecase_auth_domain.go
│   │       └── usecase_product_domain.go
│   ├── dtos
│   │   ├── product_dtos.go
│   │   ├── swagger_dtos.go
│   │   └── user_dto.go
│   ├── handler
│   │   ├── auth_handler.go
│   │   ├── middleware
│   │   │   └── middleware.go
│   │   ├── product_handler.go
│   │   └── validator
│   │       ├── product_validator.go
│   │       └── user_validator.go
│   ├── infrastructure
│   │   ├── database
│   │   │   └── db.go
│   │   ├── logger
│   │   │   └── logger.go
│   │   ├── messaging
│   │   │   └── messaging.go
│   │   └── repository
│   │       ├── product_repository.go
│   │       └── user_repository.go
│   ├── server
│   │   ├── routes.go
│   │   └── server.go
│   └── usecase
│       ├── auth_usecase.go
│       ├── product_usecase.go
│       └── test
│           ├── auth_usecase_test.go
│           └── product_usecase_test.go
├── Makefile
├── docker-compose.yml
├── go.mod
├── go.sum
├── .env
```

---

## Automação e Desenvolvimento Local

Para otimizar o fluxo de trabalho, o projeto conta com ferramentas de automação para tarefas comuns e live reload durante o desenvolvimento.

### Makefile

O `Makefile` centraliza os comandos mais utilizados, simplificando a interação com o projeto.

| Comando            | Descrição                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------- |
| `make docker-run`  | Sobe todos os containers (API, banco de dados, etc.) definidos no `docker-compose.yml`.   |
| `make docker-down` | Para e remove todos os containers da aplicação.                                           |
| `make run`         | **Inicia a API localmente em modo de desenvolvimento com live reload via `Air`.** |
| `make build`       | Compila o código fonte da API e gera o binário executável.                                |
| `make test`        | Executa a suíte de testes do projeto.                                                     |
| `make install-tools` | Instala as dependências e ferramentas de linha de comando necessárias para o projeto.   |
| `make clean`       | Remove os arquivos binários gerados pela compilação.                                      |

### Live Reload com Air

Para agilizar o desenvolvimento, este projeto utiliza a ferramenta [Air](https://github.com/air-verse/air) para live reloading.

Ao executar o comando `make run`, o Air monitora todas as alterações nos arquivos do projeto (como `.go`, `.env`, etc.). Ao detectar uma mudança, ele automaticamente recompila e reinicia o servidor da API.

Isso elimina a necessidade de parar e iniciar o servidor manualmente a cada alteração, tornando o ciclo de desenvolvimento muito mais rápido e produtivo. As configurações específicas do Air para este projeto podem ser encontradas no arquivo `.air.toml`.

### 📨 Fluxo de Notificações (Emails)

A cada operação CRUD (CREATE, UPDATE, DELETE), a API publica um evento no RabbitMQ, que é consumido pelo Consumer e envia um email de notificação.
<br>
<br>
<div align="center">
  <img src="https://i.ibb.co/qM3RRhBW/Screenshot-2025-08-31-at-15-30-27-removebg-preview.png" 
       alt="Exemplo de Email" 
       width="50%">
</div>

#### Exemplo: operação CREATE

<div align="center">
  <img src="https://i.ibb.co/xKyj8QdG/Screenshot-2025-08-31-at-15-38-35.png" 
       alt="Exemplo de Email" 
       width="50%">
</div>
