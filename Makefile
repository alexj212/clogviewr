export PROJ_PATH=github.com/alexj212/clogviewr



export DATE := $(shell date +%Y.%m.%d-%H%M)
export LATEST_COMMIT := $(shell git log --pretty=format:'%h' -n 1)
export BRANCH := $(shell git branch |grep -v "no branch"| grep \*|cut -d ' ' -f2)
export BUILT_ON_IP := $(shell [ $$(uname) = Linux ] && hostname -i || hostname )
export BIN_DIR=./bin


export BUILT_ON_OS=$(shell uname -a)
ifeq ($(BRANCH),)
BRANCH := master
endif

export COMMIT_CNT := $(shell git rev-list HEAD | wc -l | sed 's/ //g' )
export BUILD_NUMBER := ${BRANCH}-${COMMIT_CNT}

export COMPILE_LDFLAGS=-s -X "main.BuildDate=${DATE}" \
                          -X "main.LatestCommit=${LATEST_COMMIT}" \
                          -X "main.BuildNumber=${BUILD_NUMBER}" \
                          -X "main.BuiltOnIp=${BUILT_ON_IP}" \
                          -X "main.BuiltOnOs=${BUILT_ON_OS}"



build_info: check_prereq ## Build the container
	@echo ''
	@echo '---------------------------------------------------------'
	@echo 'BUILT_ON_IP       $(BUILT_ON_IP)'
	@echo 'BUILT_ON_OS       $(BUILT_ON_OS)'
	@echo 'DATE              $(DATE)'
	@echo 'LATEST_COMMIT     $(LATEST_COMMIT)'
	@echo 'BRANCH            $(BRANCH)'
	@echo 'COMMIT_CNT        $(COMMIT_CNT)'
	@echo 'BUILD_NUMBER      $(BUILD_NUMBER)'
	@echo 'COMPILE_LDFLAGS   $(COMPILE_LDFLAGS)'
	@echo 'PATH              $(PATH)'
	@echo '---------------------------------------------------------'
	@echo ''


####################################################################################################################
##
## help for each task - https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
##
####################################################################################################################
.PHONY: help

help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help



####################################################################################################################
##
## Code vetting tools
##
####################################################################################################################


test: ## run tests
	go test -v $(PROJ_PATH)

fmt: ## run fmt on project
	#go fmt $(PROJ_PATH)/...
	gofmt -s -d -w -l .

doc: ## launch godoc on port 6060
	godoc -http=:6060

deps: ## display deps for project
	go list -f '{{ join .Deps  "\n"}}' . |grep "/" | grep -v $(PROJ_PATH)| grep "\." | sort |uniq

lint: ## run lint on the project
	golint ./...

staticcheck: ## run staticcheck on the project
	staticcheck -ignore "$(shell cat .checkignore)" .

vet: ## run go vet on the project
	go vet .

reportcard: ## run goreportcard-cli
	goreportcard-cli -v

tools: ## install dependent tools for code analysis
	go install github.com/gordonklaus/ineffassign@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install golang.org/x/lint/golint@latest
	go install github.com/gojp/goreportcard/cmd/goreportcard-cli@latest
	go install github.com/goreleaser/goreleaser@latest





add_global_module: ##add dependency to global src
	@if [ ! -d "$(GOPATH)/src/$(MODULE_DIR)" ]; then \
		echo "Adding module $(MODULE_DIR)"; \
		cd $(GOPATH); \
		go get -u $(MODULE_DIR); \
	fi


add_global_binary: ##add dependency to global bin
	@if [ ! -f "$(GOPATH)/bin/$(BINARY)" ]; then \
		echo "adding binary for $(BINARY)"; \
		go install $(BINARY_URL); \
	fi




add_global_libs: ## add global libs
	@make --no-print-directory MODULE_DIR=github.com/gogo/protobuf            	add_global_module
	@make --no-print-directory MODULE_DIR=github.com/golang/protobuf           	add_global_module
	@make --no-print-directory MODULE_DIR=github.com/grpc-ecosystem/grpc-gateway add_global_module
	@make --no-print-directory MODULE_DIR=github.com/mwitkow/go-proto-validators add_global_module

	@make --no-print-directory BINARY_URL=github.com/gogo/protobuf/protoc-gen-gofast	 				 BINARY=protoc-gen-gofast		add_global_binary
	@make --no-print-directory BINARY_URL=github.com/gogo/protobuf/protoc-gen-gogo	 					 BINARY=protoc-gen-gogo			add_global_binary
	@make --no-print-directory BINARY_URL=github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway BINARY=protoc-gen-grpc-gateway	add_global_binary
	@make --no-print-directory BINARY_URL=github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger	 	 BINARY=protoc-gen-swagger		add_global_binary
	@make --no-print-directory BINARY_URL=github.com/mwitkow/go-proto-validators/protoc-gen-govalidators BINARY=protoc-gen-govalidators	add_global_binary
	@make --no-print-directory BINARY_URL=github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc        BINARY=protoc-gen-doc          add_global_binary

add_prerequisites: add_global_libs ## add all prerequisites








####################################################################################################################
##
## Build of binaries
##
####################################################################################################################

all: app logeviewr simple ## build binaries in bin dir and run tests

binaries: proxy ## build binaries in bin dir

create_dir:
	@mkdir -p $(BIN_DIR)
	@rm -f $(BIN_DIR)/web
	@ln -s ../assets $(BIN_DIR)/web


check_prereq: create_dir



build_app: create_dir
	CGO_ENABLED=0 go build -o $(BIN_DIR)/$(BIN_NAME) -a -ldflags '$(COMPILE_LDFLAGS)' $(APP_PATH)




app: build_info ## build app binary in bin dir
	@echo "build app"
	@cd  examples/app
	make BIN_NAME=app APP_PATH=. build_app
	@echo ''
	@echo ''

logeviewr: build_info ## build logeviewr binary in bin dir
	@echo "build logeviewr"
	@cd  examples/logeviewr
	make BIN_NAME=logeviewr APP_PATH=. build_app
	@echo ''
	@echo ''

simple: build_info ## build simple binary in bin dir
	@echo "build simple"
	@cd  examples/simple
	make BIN_NAME=simple APP_PATH=. build_app
	@echo ''
	@echo ''


####################################################################################################################
##
## Cleanup of binaries
##
####################################################################################################################

clean: clean_proxy  ## clean all binaries in bin dir


clean_binary: ## clean binary in bin dir
	rm -f $(BIN_DIR)/$(BIN_NAME)

clean_proxy: ## clean proxy
	make BIN_NAME=proxy clean_binary
	@rm -rf $(BIN_DIR)




