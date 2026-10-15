# Node 24 runs in a container by default; use `make check WEB_RUN=` with a local Node 24.
WEB_RUN ?= docker run --rm --network host -u $(shell id -u):$(shell id -g) -e HOME=/tmp -v $(CURDIR):/repo -w /repo/web node:24-alpine
TEST_DB_URL = postgres://axon:local-dev-only@db:5432/axon_test?sslmode=disable

.PHONY: check check-docs check-scripts check-api check-web test-integration up down teaser

## check: formatting, linting, and unit tests for API and web
check: check-docs check-scripts check-api check-web

## check-docs: the French README keeps the English one's structure
check-docs:
	@sh scripts/check-readme.sh

## check-scripts: shell scripts pass ShellCheck (warnings and errors)
check-scripts:
	docker run --rm -v $(CURDIR):/mnt -w /mnt koalaman/shellcheck:v0.11.0 -S warning $$(git ls-files '*.sh') # no spaces in script names

check-api:
	@cd api && unformatted="$$(gofmt -l .)" && if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	cd api && go vet ./... && go test ./...
	cd api && golangci-lint run ./... # install v2.13.2, the version CI uses

check-web:
	cd web && $(WEB_RUN) sh -c 'npm ci --no-fund --no-audit && npm run gen:api && npm run check && npm run lint && npm test'
	@git diff --quiet web/src/lib/api-types.ts || (echo 'web/src/lib/api-types.ts is stale: commit the regenerated file'; exit 1)

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

## teaser: record the home page teaser from the local stack's demo (AXON_DEMO=1)
teaser:
	./scripts/trailer/make-trailer.sh http://localhost:3000
