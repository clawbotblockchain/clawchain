BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')
APPNAME := clawchain

# do not override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  # if VERSION is empty, then populate it with branch name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

# Update the ldflags with the app, client & server names
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=$(APPNAME) \
	-X github.com/cosmos/cosmos-sdk/version.AppName=$(APPNAME)d \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'

##############
###  Test  ###
##############

test-unit:
	@echo Running unit tests...
	@go test -mod=readonly -v -timeout 30m ./...

test-race:
	@echo Running unit tests with race condition reporting...
	@go test -mod=readonly -v -race -timeout 30m ./...

test-cover:
	@echo Running unit tests and creating coverage report...
	@go test -mod=readonly -v -timeout 30m -coverprofile=$(COVER_FILE) -covermode=atomic ./...
	@go tool cover -html=$(COVER_FILE) -o $(COVER_HTML_FILE)
	@rm $(COVER_FILE)

bench:
	@echo Running unit tests with benchmarking...
	@go test -mod=readonly -v -timeout 30m -bench=. ./...

test: govet govulncheck test-unit

.PHONY: test test-unit test-race test-cover bench

#################
###  Install  ###
#################

all: install

install:
	@echo "--> ensure dependencies have not been modified"
	@go mod verify
	@echo "--> installing $(APPNAME)d"
	@go install $(BUILD_FLAGS) -mod=readonly ./cmd/$(APPNAME)d

.PHONY: all install

##################
###  Protobuf  ###
##################

# Use this target if you do not want to use Ignite for generating proto files

proto-deps:
	@echo "Installing proto deps"
	@echo "Proto deps present, run 'go tool' to see them"

proto-gen:
	@echo "Generating protobuf files..."
	@ignite generate proto-go --yes

.PHONY: proto-gen

#################
###  Linting  ###
#################

lint:
	@echo "--> Running linter"
	@go tool github.com/golangci/golangci-lint/cmd/golangci-lint run ./... --timeout 15m

lint-fix:
	@echo "--> Running linter and fixing issues"
	@go tool github.com/golangci/golangci-lint/cmd/golangci-lint run ./... --fix --timeout 15m

.PHONY: lint lint-fix

###################
### Development ###
###################

govet:
	@echo Running go vet...
	@go vet ./...

govulncheck:
	@echo Running govulncheck...
	@go tool golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...

.PHONY: govet govulncheck

####################
### Build Targets ##
####################

build:
	@echo "--> Building $(APPNAME)d"
	@go build $(BUILD_FLAGS) -mod=readonly -o build/$(APPNAME)d ./cmd/$(APPNAME)d

build-linux:
	@echo "--> Cross-compiling $(APPNAME)d for linux/amd64"
	@GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -mod=readonly -o build/$(APPNAME)d-linux-amd64 ./cmd/$(APPNAME)d

.PHONY: build build-linux

#####################
### Local Testnet ###
#####################

localnet-init:
	@echo "--> Building chain binary"
	@go install $(BUILD_FLAGS) -mod=readonly ./cmd/$(APPNAME)d
	@echo "--> Initializing local testnet"
	@bash scripts/localnet-init.sh

localnet-start:
	@bash scripts/localnet-start.sh

localnet-stop:
	@bash scripts/localnet-stop.sh

localnet-clean:
	@echo "--> Cleaning local testnet data"
	@rm -rf /tmp/clawchain-localnet

.PHONY: localnet-init localnet-start localnet-stop localnet-clean

####################
### Deployment   ###
####################

deploy-validator:
	@bash scripts/deploy-validator.sh

server-setup:
	@bash scripts/server-setup.sh

.PHONY: deploy-validator server-setup