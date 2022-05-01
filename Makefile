#!/usr/bin/make -f

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
PACKAGES_NOSIMULATION=$(shell go list ./... | grep -v '/simulation')
BINDIR ?= $(GOPATH)/bin
DOCKER := $(shell which docker)
DOCKER_BUF := $(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace bufbuild/buf
BUILDDIR ?= $(CURDIR)/build
SIMAPP = ./app

export GO111MODULE = on

# process build tags

build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq ($(WITH_CLEVELDB),yes)
  build_tags += gcc
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=fundraisingd \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=fundraisingd \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
		  -X github.com/cosmos/cosmos-sdk/types.reDnmString=[a-zA-Z][a-zA-Z0-9/:]{2,127}

ifeq ($(WITH_CLEVELDB),yes)
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

all: tools install lint

# The below include contains the tools and runsim targets.
include contrib/devtools/Makefile

###############################################################################
###                                  Build                                  ###
###############################################################################

build: go.sum
ifeq ($(OS),Windows_NT)
	go build $(BUILD_FLAGS) -o build/fundraisingd.exe ./cmd/fundraisingd
else
	go build $(BUILD_FLAGS) -o build/fundraisingd ./cmd/fundraisingd
endif

build-linux: go.sum
	LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build

install: go.sum
	go install $(BUILD_FLAGS) ./cmd/fundraisingd

build-reproducible: go.sum
	$(DOCKER) rm latest-build || true
	$(DOCKER) run --volume=$(CURDIR):/sources:ro \
        --env TARGET_PLATFORMS='linux/amd64 darwin/amd64 linux/arm64' \
        --env APP=fundraisingd \
        --env VERSION=$(VERSION) \
        --env COMMIT=$(COMMIT) \
        --env LEDGER_ENABLED=$(LEDGER_ENABLED) \
        --name latest-build cosmossdk/rbuilder:latest
	$(DOCKER) cp -a latest-build:/home/builder/artifacts/ $(CURDIR)/

###############################################################################
###                          Tools & Dependencies                           ###
###############################################################################

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify
	@go mod tidy

clean:
	rm -rf build/

.PHONY: go-mod-cache clean

###############################################################################
###                           Tests & Simulation                            ###
###############################################################################

test: test-unit
test-all: test-unit test-race test-cover

test-unit: 
	@VERSION=$(VERSION) go test -mod=readonly -tags='norace' $(PACKAGES_NOSIMULATION)

test-race:
	@go test -mod=readonly -timeout 30m -race -coverprofile=coverage.txt -covermode=atomic -tags='ledger test_ledger_mock' ./...

test-cover:
	@go test -mod=readonly -timeout 30m -coverprofile=coverage.txt -covermode=atomic -tags='norace ledger test_ledger_mock' ./...

.PHONY: test test-all test-unit test-race test-cover

SIM_NUM_BLOCKS ?= 100
SIM_BLOCK_SIZE ?= 50
SIM_CI_NUM_BLOCKS ?= 200
SIM_CI_BLOCK_SIZE ?= 26
SIM_PERIOD ?= 50
SIM_COMMIT ?= true
SIM_TIMEOUT ?= 24h

# test-sim-nondeterminism: Run simulation test checking for app state nondeterminism
test-sim-nondeterminism:
	@echo "Running non-determinism test..."
	@VERSION=$(VERSION) go test -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Commit=$(SIM_COMMIT) -Period=$(SIM_PERIOD)  \
		-v -timeout $(SIM_TIMEOUT)

# test-sim-import-export: Run simulation test checking import and export app state determinism
# go get github.com/cosmos/tools/cmd/runsim@v1.0.0
test-sim-import-export: runsim
	@echo "Running application import/export simulation. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 2 2 TestAppImportExport

# test-sim-after-import: Run simulation test checking import after simulation
# go get github.com/cosmos/tools/cmd/runsim@v1.0.0
test-sim-after-import: runsim
	@echo "Running application simulation-after-import. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 2 2 TestAppSimulationAfterImport

test-sim-nondeterminism-long:
	@echo "Running non-determinism test..."
	@go test -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=100 -BlockSize=100 -Commit=true -Period=0 -v -timeout 1h

test-sim-import-export-long: runsim
	@echo "Running application import/export simulation. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 5 5 TestAppImportExport

test-sim-after-import-long: runsim
	@echo "Running application simulation-after-import. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 5 5 TestAppSimulationAfterImport

# test-sim-ci: Run lightweight simulation for CI pipeline
test-sim-ci:
	@echo "Running application benchmark for numBlocks=$(SIM_CI_NUM_BLOCKS), blockSize=$(SIM_CI_BLOCK_SIZE)"
	@VERSION=$(VERSION) go test -mod=readonly -benchmem -run=^$$ $(SIMAPP) -bench ^BenchmarkSimulation$$  \
		-Enabled=true -NumBlocks=$(SIM_CI_NUM_BLOCKS) -BlockSize=$(SIM_CI_BLOCK_SIZE) -Commit=$(SIM_COMMIT) \
		-Period=$(SIM_PERIOD) -timeout $(SIM_TIMEOUT)

# test-sim-benchmark: Run heavy benchmarking simulation
test-sim-benchmark:
	@echo "Running application benchmark for numBlocks=$(SIM_NUM_BLOCKS), blockSize=$(SIM_BLOCK_SIZE). This may take awhile!"
	@VERSION=$(VERSION) go test -mod=readonly -benchmem -run=^$$ $(SIMAPP) -bench ^BenchmarkSimulation$$  \
		-Enabled=true -NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Period=$(SIM_PERIOD) \
		-Commit=$(SIM_COMMIT) timeout $(SIM_TIMEOUT)

# test-sim-benchmark: Run heavy benchmarking simulation with CPU and memory profiling
test-sim-profile:
	@echo "Running application benchmark for numBlocks=$(SIM_NUM_BLOCKS), blockSize=$(SIM_BLOCK_SIZE). This may take awhile!"
	@VERSION=$(VERSION) go test -mod=readonly -benchmem -run=^$$ $(SIMAPP) -bench ^BenchmarkSimulation$$ \
		-Enabled=true -NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Period=$(SIM_PERIOD) \
		-Commit=$(SIM_COMMIT) timeout $(SIM_TIMEOUT)-cpuprofile cpu.out -memprofile mem.out

.PHONY: \
test-sim-nondeterminism \
test-sim-nondeterminism-long \
test-sim-import-export \
test-sim-import-export-long \
test-sim-after-import \
test-sim-after-import-long \
test-sim-ci \
test-sim-profile \
test-sim-benchmark

###############################################################################
###                                Protobuf                                 ###
###############################################################################

containerProtoVer=v0.2
containerProtoImage=tendermintdev/sdk-proto-gen:$(containerProtoVer)
containerProtoGen=cosmos-sdk-proto-gen-$(containerProtoVer)
containerProtoGenSwagger=cosmos-sdk-proto-gen-swagger-$(containerProtoVer)
containerProtoFmt=cosmos-sdk-proto-fmt-$(containerProtoVer)

proto-all: proto-format proto-gen proto-swagger-gen 

proto-gen:
	starport generate proto-go

proto-swagger-gen:
	starport generate openapi

proto-format:
	@echo "Formatting Protobuf files"
	@if docker ps -a --format '{{.Names}}' | grep -Eq "^${containerProtoFmt}$$"; then docker start -a $(containerProtoFmt); else docker run --name $(containerProtoFmt) -v $(CURDIR):/workspace --workdir /workspace tendermintdev/docker-build-proto \
		find ./ -not -path "./third_party/*" -name "*.proto" -exec clang-format -i {} \; ; fi

.PHONY: proto-all proto-gen proto-swagger-gen proto-format proto-lint

###############################################################################
###                               Localnet                                  ###
###############################################################################

localnet: 
	ignite chain serve -r -v -c ./config-test.yml

.PHONY: localnet
