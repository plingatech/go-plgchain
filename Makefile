
.PHONY: download-submodules
download-submodules:
	git submodule init
	git submodule update

.PHONY: bindata
bindata:
	go-bindata -pkg chain -o ./chain/chain_bindata.go ./chain/chains

.PHONY: protoc
protoc:
	protoc --go_out=. --go-grpc_out=. -I . -I=./validate --validate_out="lang=go:." \
	 ./server/proto/*.proto \
	 ./network/proto/*.proto \
	 ./txpool/proto/*.proto	\
	 ./consensus/ibft/**/*.proto \
	 ./consensus/plgbft/**/*.proto

.PHONY: build
build:
	$(eval LATEST_VERSION = $(shell git describe --tags --abbrev=0))
	$(eval COMMIT_HASH = $(shell git rev-parse HEAD))
	$(eval BRANCH = $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n'))
	$(eval TIME = $(shell date))
	go build -o go-plgchain -ldflags="\
    	-X 'github.com/plingatech/go-plgchain/versioning.Version=$(LATEST_VERSION)' \
		-X 'github.com/plingatech/go-plgchain/versioning.Commit=$(COMMIT_HASH)'\
		-X 'github.com/plingatech/go-plgchain/versioning.Branch=$(BRANCH)'\
		-X 'github.com/plingatech/go-plgchain/versioning.BuildTime=$(TIME)'" \
	main.go

.PHONY: lint
lint:
	golangci-lint run --config .golangci.yml

.PHONY: generate-bsd-licenses
generate-bsd-licenses:
	./generate_dependency_licenses.sh BSD-3-Clause,BSD-2-Clause > ./licenses/bsd_licenses.json

.PHONY: test
test:
	go test -coverprofile coverage.out -timeout 20m `go list ./... | grep -v e2e`

.PHONY: test-e2e
test-e2e:
    # We need to build the binary with the race flag enabled
    # because it will get picked up and run during e2e tests
    # and the e2e tests should error out if any kind of race is found
	go build -race -o artifacts/go-plgchain .
	env EDGE_BINARY=${PWD}/artifacts/go-plgchain go test -v -timeout=30m ./e2e/...

.PHONY: test-e2e-plgbft
test-e2e-plgbft:
    # We can not build with race because of a bug in boltdb dependency
	go build -o artifacts/go-plgchain .
	env EDGE_BINARY=${PWD}/artifacts/go-plgchain E2E_TESTS=true E2E_LOGS=true E2E_TESTS_TYPE=integration \
	go test -v -timeout=45m ./e2e-plgbft/e2e/...

test-property-plgbft:
    # We can not build with race because of a bug in boltdb dependency
	go build -o artifacts/go-plgchain .
	env EDGE_BINARY=${PWD}/artifacts/go-plgchain E2E_TESTS=true E2E_LOGS=true E2E_TESTS_TYPE=property go test -v -timeout=30m ./e2e-plgbft/property/...

.PHONY: compile-core-contracts
compile-core-contracts:
	cd core-contracts && npm install && npm run compile
	$(MAKE) generate-smart-contract-bindings

.PHONY: generate-smart-contract-bindings
generate-smart-contract-bindings:
	go run ./consensus/plgbft/contractsapi/artifacts-gen/main.go
	go run ./consensus/plgbft/contractsapi/bindings-gen/main.go
