.PHONY: build test lint typecheck format docker-up docker-down docker-logs

# Go API
GO_APP := apps/api
DIFF_SERVICE := apps/diff-service
WEB := apps/web

build:
	cd $(GO_APP) && go build ./cmd/server
	cd $(DIFF_SERVICE) && bun run build
	cd $(WEB) && npm run build

test:
	cd $(GO_APP) && go test ./...
	cd $(DIFF_SERVICE) && bun test
	cd $(WEB) && npm test
	cd packages/diff-engine && npm test
	cd packages/sdk-typescript && npm test

lint:
	cd $(GO_APP) && go vet ./...
	cd $(DIFF_SERVICE) && eslint src --ext .ts
	cd $(WEB) && npm run lint
	cd packages/diff-engine && eslint src --ext .ts
	cd packages/sdk-typescript && eslint src --ext .ts

typecheck:
	cd $(DIFF_SERVICE) && tsc --noEmit
	cd $(WEB) && npm run typecheck
	cd packages/diff-engine && tsc --noEmit
	cd packages/sdk-typescript && tsc --noEmit

format:
	cd $(DIFF_SERVICE) && prettier --write src
	cd $(WEB) && prettier --write .
	cd packages/diff-engine && prettier --write src
	cd packages/sdk-typescript && prettier --write src

docker-build:
	docker compose -f deploy/docker-compose.yml build

docker-up:
	docker compose -f deploy/docker-compose.yml up -d

docker-down:
	docker compose -f deploy/docker-compose.yml down

docker-logs:
	docker compose -f deploy/docker-compose.yml logs -f

install:
	pnpm install
	cd $(GO_APP) && go mod tidy
	cd packages/sdk-python && uv pip install -e ".[dev]"

clean:
	rm -rf node_modules
	rm -rf packages/*/node_modules
	rm -rf apps/*/node_modules
	rm -rf apps/api/data
	rm -rf deploy/data