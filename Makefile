.PHONY: dev dev-backend dev-frontend build test generate lint

dev:
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	cd backend && TIPSARR_CONFIG_DIR=./config go run ./cmd/tipsarr

dev-frontend:
	cd frontend && npm run dev

build:
	docker build -t tipsarr:dev .

test:
	cd backend && go test ./...
	cd frontend && npm run check

# Re-export the OpenAPI spec and regenerate the frontend TypeScript types.
generate:
	cd backend && go run ./cmd/tipsarr openapi > ../frontend/src/lib/api/openapi.json
	cd frontend && npx openapi-typescript@7.13.0 src/lib/api/openapi.json -o src/lib/api/schema.d.ts

lint:
	cd backend && go vet ./...
