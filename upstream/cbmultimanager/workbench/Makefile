EXECUTABLES = openssl sqlcipher go npm git
OK := $(foreach exec,$(EXECUTABLES),\
	$(if $(shell which $(exec)),do nothing,$(error "No $(exec) in PATH")))

OPEN_SSL_LIB = -L/usr/local/opt/openssl@1.1/lib
OPEN_SSL_INCLUDE = -I/usr/local/opt/openssl@1.1/include
PWD = $(shell pwd)
BUILD_DIR = $(PWD)/build
CERT_DIR = $(BUILD_DIR)/cert
UI_DIR = $(BUILD_DIR)/ui
DATA_DIR = $(BUILD_DIR)/data
LOGS_DIR = $(BUILD_DIR)/logs
CBMM_SOURCE_DIR = $(PWD)/cbmultimanager
CBMM_UI_MN_MODULES_SRC_DIR = $(CBMM_SOURCE_DIR)/ui/mn-modules

all:
	make -j2 build-and-watch-ui build-and-run-server

create-dir-build:
	mkdir -p $(BUILD_DIR)

create-dir-cert: create-dir-build
	mkdir -p $(CERT_DIR)

create-dir-ui: create-dir-build
	mkdir -p $(UI_DIR)

create-dir-data: create-dir-build
	mkdir -p $(DATA_DIR)

create-dir-log: create-dir-build
	mkdir -p $(LOGS_DIR)

create-dir-mn-modules:
	mkdir -p $(CBMM_UI_MN_MODULES_SRC_DIR)

generate-certificates: create-dir-cert
	openssl req -x509 -newkey rsa:4096 \
	-keyout $(CERT_DIR)/key.pem \
	-out $(CERT_DIR)/cert.pem \
	-nodes -days 365 \
	-subj "/C=US/ST=Oregon/L=Portland/O=Company Name/OU=Org/CN=www.example.com"

copy-mn-modules-src: create-dir-mn-modules
	cp -R ns_server/priv/public/ui/ $(CBMM_UI_MN_MODULES_SRC_DIR)

build-cbmultimanager: create-dir-build
	cd $(CBMM_SOURCE_DIR); \
	CGO_ENABLED=1 \
	CGO_LDFLAGS=$(OPEN_SSL_LIB) \
	CGO_CPPFLAGS=$(OPEN_SSL_INCLUDE) \
	CGO_CFLAGS=$(OPEN_SSL_INCLUDE) \
	CGO_CXXFLAGS=$(OPEN_SSL_INCLUDE) \
	go build -o $(BUILD_DIR) ./cmd/cbmultimanager

build-and-watch-ui: create-dir-ui copy-mn-modules-src
	cd $(CBMM_SOURCE_DIR)/ui; \
	npm install; npm run watch -- --output-path=$(UI_DIR)
.PHONY: build-and-watch-ui

build-and-run-server: create-dir-data create-dir-log generate-certificates build-cbmultimanager
	$(BUILD_DIR)/cbmultimanager \
	--cert-path $(CERT_DIR)/cert.pem \
	--key-path $(CERT_DIR)/key.pem \
	--sqlite-db $(DATA_DIR)/data.sqlite \
	--sqlite-key password \
	--log-level debug \
	--ui-root $(UI_DIR) \
	--log-dir $(LOGS_DIR)
.PHONY: build-and-run-server

clean:
	-(rm -rf $(BUILD_DIR))
	-(cd $(CBMM_SOURCE_DIR); git clean -dfxq)
