# Node 24 runs in a container by default; use `make check WEB_RUN=` with a local Node 24.
WEB_RUN ?= docker run --rm --network host -u $(shell id -u):$(shell id -g) -e HOME=/tmp -v $(CURDIR)/web:/app -w /app node:24-alpine
TEST_DB_URL = postgres://axon:local-dev-only@db:5432/axon_test?sslmode=disable

.PHONY: check check-api check-web test-integration up down

## check: formatting, linting, and unit tests for API and web
check: check-api check-web

check-api:
	@cd api && unformatted="$$(gofmt -l .)" && if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	cd api && go vet ./... && go test ./...
	docker run --rm --network host -v $(CURDIR)/api:/src -v axon-gomod:/go/pkg/mod -w /src golangci/golangci-lint:v2.13.2 golangci-lint run ./...

check-web:
	cd web && $(WEB_RUN) sh -c 'npm ci --no-fund --no-audit && npm run check && npm run lint && npm test'

## test-integration: API and migration tests against a real PostgreSQL + pgvector
test-integration:
	docker compose up -d --wait db
	docker compose exec db sh -c 'createdb -U "$$POSTGRES_USER" axon_test 2>/dev/null || true'
	docker run --rm --network axon_default -e AXON_TEST_DATABASE_URL="$(TEST_DB_URL)" \
		-v $(CURDIR)/api:/src -v axon-gomod:/go/pkg/mod -w /src golang:1.26-alpine \
		go test -count=1 -p 1 ./...

up:
	docker compose up -d --build --wait

down:
	docker compose down
