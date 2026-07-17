.PHONY: build run dev test lint swagger clean docker-up docker-down

# ── Build ──────────────────────────────────────────────────────────────
build:
	go build -o bin/netlab-server .

# ── Run ────────────────────────────────────────────────────────────────
run: build
	./bin/netlab-server

# ── Development (with hot-reload via air) ───────────────────────────────
dev:
	air

# ── Test ────────────────────────────────────────────────────────────────
test:
	go test ./... -v -cover

test-race:
	go test ./... -v -race

# ── Lint ────────────────────────────────────────────────────────────────
lint:
	golangci-lint run ./...

# ── Swagger Documentation ───────────────────────────────────────────────
swagger:
	swag init -g main.go --output docs

# ── Docker ──────────────────────────────────────────────────────────────
docker-up:
	docker-compose up -d postgres redis

docker-down:
	docker-compose down

# ── Clean ───────────────────────────────────────────────────────────────
clean:
	rm -rf bin/
	rm -rf docs/
