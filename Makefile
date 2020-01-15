.PHONY: build
## build: builds server
build:
	@cd app && go build -v -mod=vendor

.PHONY: vet
## vet: runs `go vet`
vet:
	@cd app && go vet ./...

.PHONY: vendor
## vendor: runs `go mod vendor`
vendor:
	@go mod vendor

.PHONY: test
## test: runs `go test`
test:
	@cd app && go test -mod=vendor ./... -coverprofile cover.out
	
.PHONY: lint
## lint: runs `golint`
lint:
	@cd app && golint ./...

.PHONY: run
## run: runs app locally (don't forget to set all required environment variables)
# examples:
# make run ARGS="--super umputun"
# make run ARGS="--super umputun --export-num 684 --export-path logs --export-day 20200104 --export-template data/logs.html"
run:
	@go run -v -mod=vendor app/main.go ${ARGS}

.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
