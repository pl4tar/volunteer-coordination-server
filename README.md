# Volunteer Coordination Server

Backend service for coordinating volunteers and related workflows (API + persistence), written in Go.

---

## Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Configuration](#configuration)
    - [Run locally](#run-locally)
    - [Run with Docker](#run-with-docker)
- [API Documentation (Swagger)](#api-documentation-swagger)
- [Database](#database)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Security](#security)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

This repository contains a Go server that exposes an HTTP API for volunteer coordination scenarios. It supports running locally and via Docker Compose with PostgreSQL.

> If you’re evaluating the project: start it with Docker Compose, then open Swagger UI and explore the endpoints.

---

## Tech Stack

- **Go** `1.25.x`
- **PostgreSQL** (via Docker image)
- **Docker / Docker Compose**
- **Swagger / OpenAPI** via `swaggo/swag`

---

## Project Structure

Common Go project layout:

- `cmd/` — application entrypoints (the main binary)
- `internal/` — private application code (business logic, storage, handlers, etc.)
- `config/` — configuration files (e.g., local/dev)
- `docs/` — additional documentation
- `diagrams/` — architecture/flow diagrams
- `Dockerfile`, `docker-compose.yml` — containerization

---

## Getting Started

### Prerequisites

- Go `1.25.x`
- Docker + Docker Compose (recommended for quickest start)
- (Optional) `swag` CLI if you want to generate Swagger docs locally

### Configuration

The app is configured via **environment variables** (and a config file path).

Typical variables:

- `API_PORT` — port the HTTP server listens on
- `CONFIG_PATH` — path to the YAML config inside/outside container
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD` — PostgreSQL connection

#### Environment file

If there is no `.env.example` yet, create it and keep real credentials only in `.env`.
```
CONFIG_PATH=./config/local.yaml;
DB_HOST=localhost;
DB_NAME=volonteer;
DB_PASSWORD=8967451230;
DB_PORT=5432;
DB_USERNAME=postgres
```
### Run with Docker
```aiignore
docker compose up --build -d
```

## API Documentation (Swagger)

This project uses `swaggo/swag` to generate Swagger/OpenAPI docs.

### Install tools
```
go install github.com/swaggo/swag/cmd/swag@latest
go get github.com/swaggo/gin-swagger
go get github.com/swaggo/files
```
```
swag --version
swag init -g cmd/app/main.go

```


