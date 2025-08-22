# GO REDIS LUA SCRIPT (GRLS)

GO REDIS LUA SCRIPT (GRLS) is app for try for not race condition

# Project Structure

```
grls
├─ .air.toml
├─ Makefile
├─ README.md
├─ cmd
│  └─ server
│     └─ main.go
├─ deploy
│  └─ docker
│     ├─ Dockerfile
│     └─ docker-compose.yml
├─ go.mod
├─ go.sum
├─ internal
│  ├─ app
│  │  ├─ app.go
│  │  ├─ factory
│  │  │  └─ factory.go
│  │  └─ routes
│  │     ├─ healthz_routes.go
│  │     └─ routes.go
│  ├─ config
│  │  └─ config.go
│  ├─ infrastructure
│  │  ├─ db
│  │  │  └─ db.go
│  │  └─ repository
│  │     └─ .keep
│  └─ modules
│     └─ point
│        ├─ dto
│        │  └─ .keep
│        ├─ handler
│        │  └─ .keep
│        ├─ model
│        │  └─ .keep
│        └─ usecase
│           └─ .keep
├─ migration
│  └─ .keep
└─ pkg
   ├─ logger
   │  └─ logger.go
   └─ response
      └─ response.go

```

# Prerequisites

Before starting, ensure you have the following installed:

- Go (version 1.20 or later)
- Docker
- Docker Compose
- golang-migrate

# Installation

1. Clone the repository

```
git clone https://github.com/beyouli/grls.git // https
git clone git@github.com:beyouli/grls.git // ssh
cd grls
```

2. Install Dependencies Ensure all Go dependencies are installed:

```
go mod tidy
```

3. Install golang-migrate Download and install the golang-migrate binary:

```
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.15.2/migrate.darwin-amd64.tar.gz | tar xvz
mv migrate ~/go/bin/migrate
chmod +x ~/go/bin/migrate
```

Verify the binary supports PostgreSQL:

```
migrate -version
```

# Configuration

create a .env file in the project root with the following file .env.example

# Running APP

## Run Application in Development

1. Use the following command to build and run the project in development mode:

```
make docker-up
```

2. To down the containers:

```
make docker-down
```

## Run Database Migrations

Use the following commands to manage database migrations:

- Create a New Migration File:

```
make migrate-create
```

- Run Migrations:

```
make migrate-up
```

- Rollback a Migration:

```
make migrate-down
```

- Rollback All Migrations:

```
make migrate-down-all
```

```
go-redis-lua_script
├─ .air.toml
├─ Makefile
├─ README.md
├─ cmd
│  ├─ server
│  │  └─ main.go
│  └─ sim_deposit
│     └─ main.go
├─ deploy
│  ├─ docker
│  │  ├─ Dockerfile
│  │  └─ docker-compose.yml
│  └─ k8s
│     └─ Dockerfile
├─ go.mod
├─ go.sum
├─ internal
│  ├─ app
│  │  ├─ app.go
│  │  ├─ factory
│  │  │  ├─ factory.go
│  │  │  └─ wallet_factory.go
│  │  └─ routes
│  │     ├─ healthz_routes.go
│  │     ├─ routes.go
│  │     └─ wallet_routes.go
│  ├─ config
│  │  └─ config.go
│  ├─ infrastructure
│  │  ├─ cache
│  │  │  └─ cache.go
│  │  ├─ db
│  │  │  └─ db.go
│  │  └─ repository
│  │     └─ wallet_repository.go
│  └─ modules
│     └─ wallet
│        ├─ handler
│        │  └─ wallet_handler.go
│        ├─ model
│        │  └─ wallet_model.go
│        └─ usecase
│           └─ wallet_usecase.go
├─ migration
│  ├─ 000001_create_wallets_table.down.sql
│  └─ 000001_create_wallets_table.up.sql
└─ pkg
   ├─ graceful
   │  └─ graceful.go
   ├─ helper
   │  └─ helper.go
   ├─ logger
   │  ├─ logfile.go
   │  └─ logger.go
   ├─ response
   │  └─ response.go
   └─ validation
      ├─ default_rule.go
      └─ validation.go

```
