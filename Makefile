GO ?= go
NPM ?= npm
ADDR ?= :18080
FILE_ROOT ?= /srv/files
MIGRATIONS_DIR ?= internal/migrations

.PHONY: test
test:
	$(GO) test ./...

.PHONY: web-build
web-build:
	cd web && $(NPM) install && $(NPM) run build

.PHONY: build
build: web-build
	$(GO) build -buildvcs=false ./cmd/server ./cmd/releasectl ./cmd/migrate

.PHONY: run
run:
	ADDR=$(ADDR) FILE_ROOT=$(FILE_ROOT) $(GO) run -buildvcs=false ./cmd/server

.PHONY: migrate
migrate:
	$(GO) run -buildvcs=false ./cmd/migrate -dir $(MIGRATIONS_DIR)

.PHONY: resource-catalog
resource-catalog:
	$(GO) run ./cmd/releasectl resource-catalog

.PHONY: smoke-db
smoke-db: web-build
	GO=$(GO) scripts/smoke_release_center_db.sh

.PHONY: resource-pack
resource-pack:
	$(GO) run ./cmd/releasectl resource-pack \
		-version $(RESOURCE_VERSION) \
		-channel $(or $(CHANNEL),dev) \
		-title "$(or $(RESOURCE_TITLE),运行资源更新)" \
		-root $(RESOURCE_ROOT) \
		-out $(or $(RESOURCE_OUT),dist/app-resources) \
		-manifest-private-key "$(MANIFEST_PRIVATE_KEY)"
