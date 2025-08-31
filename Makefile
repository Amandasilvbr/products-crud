-include .env
export

.PHONY: all build run test clean install-tools docker-run docker-down air-build docs

all: build

build: docs
	@echo "Building..."
	@go build -o main cmd/api/main.go

run:
	@air

air-build: docs
	@go build -o ./tmp/main cmd/api/main.go

docs: cmd/api/docs/docs.go

cmd/api/docs/docs.go: $(shell find internal/handler internal/server -type f)
	@swag fmt -d ./internal/handler,./internal/server
	@swag init --parseDependency --parseInternal -d ./cmd/api,./internal/handler,./internal/server -o ./cmd/api/docs

docker-run:
	@echo "Starting Docker Compose..."
	@docker compose up -d 2>/dev/null || (echo "Falling back to Docker Compose V1"; docker-compose up -d)

docker-down:
	@echo "Stopping Docker Compose..."
	@docker compose down 2>/dev/null || (echo "Falling back to Docker Compose V1"; docker-compose down)

install-tools:
	@go install github.com/swaggo/swag/cmd/swag@v1.16.3
	@go install github.com/air-verse/air@v1.52.3
	@go get -u gorm.io/gorm
	@go get -u gorm.io/driver/postgres
	@go get -u github.com/joho/godotenv
# 	@go get gorm.io/datatypes
	@go get github.com/go-playground/validator/v10
	@go get github.com/dgrijalva/jwt-go
	@go get -u go.uber.org/zap
	@go get -u github.com/swaggo/swag
	@go get -u github.com/swaggo/gin-swagger
	@go get -u github.com/swaggo/files
	@go get -u github.com/rabbitmq/amqp091-go
	@go get -u github.com/jordan-wright/email
	@go get -u github.com/rabbitmq/amqp091-go
	@go get -u github.com/stretchr/testify/assert
	@go get -u github.com/stretchr/testify/mock
	@go get -u gotest.tools/gotestsum@latest


test:
	@echo "Running tests..."
	@go test ./tests/... -v

clean:
	@echo "Cleaning..."
	@rm -f main ./tmp/main