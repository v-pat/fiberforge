package templates

// DockerfileTemplate renders a multi-stage Dockerfile.
const DockerfileTemplate = `# ---- Build stage ----
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/{{.AppName}} .

# ---- Runtime stage ----
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /bin/{{.AppName}} .
EXPOSE {{.Port}}
ENTRYPOINT ["/app/{{.AppName}}"]
`

// DockerComposeTemplate renders docker-compose with the database service.
const DockerComposeTemplate = `version: "3.9"

services:
  {{.AppName}}:
    build: .
    ports:
      - "{{.Port}}:{{.Port}}"
    environment:
      DB_HOST: {{.DbHost}}
      DB_PORT: "{{.DbPort}}"
      DB_USER: {{.DbUser}}
      DB_PASSWORD: {{.DbPassword}}
      DB_NAME: {{.AppName}}
      PORT: "{{.Port}}"
      {{if .Auth}}JWT_SECRET: change-me-in-production
      {{end}}
    depends_on:
      db:
        condition: service_healthy
{{if eq .DbType "postgres"}}
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: {{.DbUser}}
      POSTGRES_PASSWORD: {{.DbPassword}}
      POSTGRES_DB: {{.AppName}}
    ports:
      - "{{.DbPort}}:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U {{.DbUser}}"]
      interval: 5s
      timeout: 5s
      retries: 5
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
{{else if eq .DbType "mysql"}}
  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: {{.DbPassword}}
      MYSQL_DATABASE: {{.AppName}}
    ports:
      - "{{.DbPort}}:3306"
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p{{.DbPassword}}"]
      interval: 5s
      timeout: 5s
      retries: 5
    volumes:
      - mysqldata:/var/lib/mysql
volumes:
  mysqldata:
{{else if eq .DbType "mongodb"}}
  db:
    image: mongo:7
    ports:
      - "{{.DbPort}}:27017"
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 5s
      timeout: 5s
      retries: 5
    volumes:
      - mongodata:/data/db
volumes:
  mongodata:
{{end}}`

// MakefileTemplate renders common project tasks.
const MakefileTemplate = `.PHONY: run build test migrate docker

run:
	go run .

build:
	go build -o bin/{{.AppName}} .

test:
	go test ./...

{{if .Migrations}}# Requires the golang-migrate CLI (install: brew install golang-migrate)
migrate:
	migrate -path migrations -database "$$DATABASE_URL" up
{{end}}{{if .Docker}}docker:
	docker compose up --build
{{end}}`

// CITemplate renders a GitHub Actions workflow.
const CITemplate = `name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.60.3
      - run: go build ./...
      - run: go vet ./...
      - run: go test -race -coverprofile=coverage.out ./...
{{if .Docker}}
  image:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/build-push-action@v6
        with:
          push: false
          tags: {{.AppName}}:latest
{{end}}`
