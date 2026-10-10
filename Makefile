.PHONY: dev dev-backend dev-frontend build test generate lint ios-strings locales-check

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
	cd frontend && npm run check && npm run check:i18n

# Re-export the OpenAPI spec and regenerate the frontend TypeScript types.
generate:
	cd backend && go run ./cmd/tipsarr openapi > ../frontend/src/lib/api/openapi.json
	mkdir -p api && cd backend && go run ./cmd/tipsarr openapi yaml > ../api/openapi.yaml
	cp api/openapi.yaml ios/Tipsarr/Tipsarr/Core/API/openapi.yaml
	cp api/openapi.yaml android/app/src/main/openapi/openapi.yaml
	node design/build-tokens.mjs
	cp design/generated/Tokens.swift ios/Tipsarr/Tipsarr/DesignSystem/Tokens.swift
	mkdir -p android/app/src/main/java/com/brebond/tipsarr/design android/app/src/main/openapi
	cp design/generated/Tokens.kt android/app/src/main/java/com/brebond/tipsarr/design/Tokens.kt
	node design/build-strings.mjs
	for d in design/generated/android/values*; do mkdir -p android/app/src/main/res/$$(basename $$d) && cp $$d/strings.xml android/app/src/main/res/$$(basename $$d)/strings.xml; done
	cp design/generated/android/GeneratedLanguages.kt android/app/src/main/java/com/brebond/tipsarr/core/support/GeneratedLanguages.kt
	cd frontend && npx openapi-typescript@7.13.0 src/lib/api/openapi.json -o src/lib/api/schema.d.ts

lint:
	cd backend && go vet ./...

# Xcode reformats the catalog it owns, so it is copied on demand, not by `generate` and not checked in CI.
ios-strings:
	cp design/generated/Localizable.xcstrings ios/Tipsarr/Tipsarr/Localizable.xcstrings
	cp design/generated/InfoPlist.xcstrings ios/Tipsarr/Tipsarr/InfoPlist.xcstrings
	node design/ios-regions.mjs

# Lists every language in locales/ and fails on a real problem (unknown key, changed placeholder). Missing translations are only listed.
locales-check:
	node design/check-locales.mjs
