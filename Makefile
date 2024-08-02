LOCAL_BIN:=$(CURDIR)/bin

COMPOSE_FILE=deployments/docker-compose.yml
COMPOSE_FILE_TEST=deployments/docker-compose-test.yml

MIGRATE_SCRIPT_TEST=./scripts/migration-test.sh
CLEAN_DB_SCRIPT_TEST=./scripts/clean-db-test.sh

MOCKGEN_BIN= ""
MOCKGEN_TAG=1.2.0

MIGRATE_OLD_ORDERS=./cmd/migrator/main.go
APP=./cmd/pvz_manager/main.go
CLEANUP=./cmd/cleanup/main.go

PROTOC := PATH="$$PATH:$(LOCAL_BIN)" protoc
ORDER_PROTO_PATH := api/proto/order/v1
VENDOR_PROTO_DIR := vendor.proto

# Установка всех необходимых зависимостей
.PHONY: .bin-deps
.bin-deps:
	$(info Installing binary dependencies...)
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	GOBIN=$(LOCAL_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0
	GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.20.0
	GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.20.0
	GOBIN=$(LOCAL_BIN) go install github.com/envoyproxy/protoc-gen-validate@v1.0.4


# Вендоринг внешних proto файлов
.vendor-proto: $(VENDOR_PROTO_DIR)/google/protobuf $(VENDOR_PROTO_DIR)/google/api $(VENDOR_PROTO_DIR)/validate $(VENDOR_PROTO_DIR)/protoc-gen-openapiv2/options

vendor.proto/protoc-gen-openapiv2/options:
	rm -rf $(VENDOR_PROTO_DIR)/grpc-ecosystem
	git clone -b main --single-branch --depth=1 --filter=tree:0 \
		https://github.com/grpc-ecosystem/grpc-gateway $(VENDOR_PROTO_DIR)/grpc-ecosystem
	cd $(VENDOR_PROTO_DIR)/grpc-ecosystem && \
		git sparse-checkout set --no-cone protoc-gen-openapiv2/options && \
		git checkout main
	mkdir -p $(VENDOR_PROTO_DIR)/protoc-gen-openapiv2
	mv $(VENDOR_PROTO_DIR)/grpc-ecosystem/protoc-gen-openapiv2/options $(VENDOR_PROTO_DIR)/protoc-gen-openapiv2
	rm -rf $(VENDOR_PROTO_DIR)/grpc-ecosystem

vendor.proto/google/protobuf:
	rm -rf $(VENDOR_PROTO_DIR)/protobuf
	git clone -b main --single-branch --depth=1 --filter=tree:0 \
		https://github.com/protocolbuffers/protobuf $(VENDOR_PROTO_DIR)/protobuf
	cd $(VENDOR_PROTO_DIR)/protobuf && \
		git sparse-checkout set --no-cone src/google/protobuf && \
		git checkout main
	mkdir -p $(VENDOR_PROTO_DIR)/google
	mv $(VENDOR_PROTO_DIR)/protobuf/src/google/protobuf $(VENDOR_PROTO_DIR)/google
	rm -rf $(VENDOR_PROTO_DIR)/protobuf

vendor.proto/google/api:
	rm -rf $(VENDOR_PROTO_DIR)/googleapis
	git clone -b master --single-branch --depth=1 --filter=tree:0 \
		https://github.com/googleapis/googleapis $(VENDOR_PROTO_DIR)/googleapis
	cd $(VENDOR_PROTO_DIR)/googleapis && \
		git sparse-checkout set --no-cone google/api && \
		git checkout master
	mkdir -p $(VENDOR_PROTO_DIR)/google
	mv $(VENDOR_PROTO_DIR)/googleapis/google/api $(VENDOR_PROTO_DIR)/google
	rm -rf $(VENDOR_PROTO_DIR)/googleapis

vendor.proto/validate:
	rm -rf $(VENDOR_PROTO_DIR)/tmp
	git clone -b main --single-branch --depth=2 --filter=tree:0 \
		https://github.com/bufbuild/protoc-gen-validate $(VENDOR_PROTO_DIR)/tmp
	cd $(VENDOR_PROTO_DIR)/tmp && \
		git sparse-checkout set --no-cone validate && \
		git checkout main
	mkdir -p $(VENDOR_PROTO_DIR)/validate
	mv $(VENDOR_PROTO_DIR)/tmp/validate $(VENDOR_PROTO_DIR)/
	rm -rf $(VENDOR_PROTO_DIR)/tmp

# Генерация proto файлов
.PHONY: generate
generate: .bin-deps .vendor-proto
	mkdir -p pkg/$(ORDER_PROTO_PATH)
	$(PROTOC) \
		-I api/proto \
		-I $(VENDOR_PROTO_DIR) \
		$(ORDER_PROTO_PATH)/order.proto \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go --go_out=./pkg/$(ORDER_PROTO_PATH) --go_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc --go-grpc_out=./pkg/$(ORDER_PROTO_PATH) --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-grpc-gateway=$(LOCAL_BIN)/protoc-gen-grpc-gateway --grpc-gateway_out=./pkg/$(ORDER_PROTO_PATH) --grpc-gateway_opt=paths=source_relative,generate_unbound_methods=true \
		--plugin=protoc-gen-openapiv2=$(LOCAL_BIN)/protoc-gen-openapiv2 --openapiv2_out=./pkg/$(ORDER_PROTO_PATH) \
		--plugin=protoc-gen-validate=$(LOCAL_BIN)/protoc-gen-validate --validate_out="lang=go,paths=source_relative:pkg/$(ORDER_PROTO_PATH)"

# Запуск приложения
app:
	go run $(APP)

# Очистка базы данных от старых заказов
cleanup:
	go run $(CLEANUP)

# Управление приложением и БД
up-all:
	docker compose -f $(COMPOSE_FILE) up -d postgres app

down:
	docker compose -f $(COMPOSE_FILE) down

# Управление БД
up-db:
	docker compose -f $(COMPOSE_FILE) up -d postgres

stop-db:
	docker compose -f $(COMPOSE_FILE) stop postgres

start-db:
	docker compose -f $(COMPOSE_FILE) start postgres

down-db:
	docker compose -f $(COMPOSE_FILE) down postgres

# Запуск тестового окружения
up-test-db:
	docker compose -f $(COMPOSE_FILE_TEST) up -d

# Очищение базы от тестовых данных
clean-test-db:
	$(CLEAN_DB_SCRIPT_TEST)

# Запуск скрипта миграций
migrate-test:
	$(MIGRATE_SCRIPT_TEST)

# Запуск интеграционных тестов
integration-test:
	go test -tags=integration -v ./...

# Запуск Unit-тестов
unit-test:
	go test -v ./...

unit-test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

# Установкой зависимости mockgen
generate-mockgen-deps:
ifeq ($(wildcard $(MOCKGEN_BIN)),)
	@GOBIN=$(LOCAL_BIN) go install go.uber.org/mock/mockgen@$(MOCKGEN_TAG)
endif

generate-mockgen:
	PATH="$(LOCAL_BIN):$$PATH" go generate -x -run=mockgen ./...

