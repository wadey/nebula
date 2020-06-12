NEBULA_CMD_PATH = "./cmd/nebula"
BUILD_NUMBER ?= dev+$(shell date -u '+%Y%m%d%H%M%S')
GO111MODULE = on
export GO111MODULE

ALL_LINUX = linux-amd64 \
	linux-386 \
	linux-ppc64le \
	linux-arm-5 \
	linux-arm-6 \
	linux-arm-7 \
	linux-arm64 \
	linux-mips \
	linux-mipsle \
	linux-mips64 \
	linux-mips64le

ALL = $(ALL_LINUX) \
	darwin-amd64 \
	freebsd-amd64 \
	windows-amd64

DOCKER_BIN = build/linux-amd64/nebula build/linux-amd64/nebula-cert

all: $(ALL:%=build/%/nebula) $(ALL:%=build/%/nebula-cert)

release: $(ALL:%=build/nebula-%.tar.gz)

release-linux: $(ALL_LINUX:%=build/nebula-%.tar.gz)

release-freebsd: build/nebula-freebsd-amd64.tar.gz

BUILD_ARGS = -trimpath

bin-windows: build/windows-amd64/nebula.exe build/windows-amd64/nebula-cert.exe
	mv $? .

bin-darwin: build/darwin-amd64/nebula build/darwin-amd64/nebula-cert
	mv $? .

bin-freebsd: build/freebsd-amd64/nebula build/freebsd-amd64/nebula-cert
	mv $? .

bin:
	go build $(BUILD_ARGS) -ldflags "-X main.Build=$(BUILD_NUMBER)" -o ./nebula ${NEBULA_CMD_PATH}
	go build $(BUILD_ARGS) -ldflags "-X main.Build=$(BUILD_NUMBER)" -o ./nebula-cert ./cmd/nebula-cert

install:
	go install $(BUILD_ARGS) -ldflags "-X main.Build=$(BUILD_NUMBER)" ${NEBULA_CMD_PATH}
	go install $(BUILD_ARGS) -ldflags "-X main.Build=$(BUILD_NUMBER)" ./cmd/nebula-cert

build/%/nebula: .FORCE
	GOOS=$(firstword $(subst -, , $*)) \
		GOARCH=$(word 2, $(subst -, ,$*)) \
		GOARM=$(word 3, $(subst -, ,$*)) \
		go build $(BUILD_ARGS) -o $@ -ldflags "-X main.Build=$(BUILD_NUMBER)" ${NEBULA_CMD_PATH}

build/%/nebula-cert: .FORCE
	GOOS=$(firstword $(subst -, , $*)) \
		GOARCH=$(word 2, $(subst -, ,$*)) \
		GOARM=$(word 3, $(subst -, ,$*)) \
		go build $(BUILD_ARGS) -o $@ -ldflags "-X main.Build=$(BUILD_NUMBER)" ./cmd/nebula-cert

build/%/nebula.exe: build/%/nebula
	mv $< $@

build/%/nebula-cert.exe: build/%/nebula-cert
	mv $< $@

build/nebula-%.tar.gz: build/%/nebula build/%/nebula-cert
	tar -zcv -C build/$* -f $@ nebula nebula-cert

build/nebula-%.zip: build/%/nebula.exe build/%/nebula-cert.exe
	cd build/$* && zip ../nebula-$*.zip nebula.exe nebula-cert.exe

vet:
	go vet -v ./...

test:
	go test -v ./...

test-cov-html:
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out

bench:
	go test -bench=.

bench-cpu:
	go test -bench=. -benchtime=5s -cpuprofile=cpu.pprof
	go tool pprof go-audit.test cpu.pprof

bench-cpu-long:
	go test -bench=. -benchtime=60s -cpuprofile=cpu.pprof
	go tool pprof go-audit.test cpu.pprof

proto: nebula.pb.go cert/cert.pb.go

nebula.pb.go: nebula.proto .FORCE
	go build github.com/golang/protobuf/protoc-gen-go
	PATH="$(PWD):$(PATH)" protoc --go_out=. $<
	rm protoc-gen-go

cert/cert.pb.go: cert/cert.proto .FORCE
	$(MAKE) -C cert cert.pb.go

service:
	@echo > /dev/null
	$(eval NEBULA_CMD_PATH := "./cmd/nebula-service")
ifeq ($(words $(MAKECMDGOALS)),1)
	$(MAKE) service ${.DEFAULT_GOAL} --no-print-directory
endif

smoke-docker: $(DOCKER_BIN)
	cd .github/workflows/smoke/ && ./build.sh
	cd .github/workflows/smoke/ && ./smoke.sh

smoke-docker-race: BUILD_ARGS = -race
smoke-docker-race: smoke-docker

smoke-vagrant/%: $(DOCKER_BIN) build/%/nebula
	cd .github/workflows/smoke/ && ./build.sh $*
	cd .github/workflows/smoke/ && ./smoke-vagrant.sh $*

.FORCE:
.PHONY: test test-cov-html bench bench-cpu bench-cpu-long bin proto release service smoke-docker smoke-docker-race smoke-vagrant/%
.DEFAULT_GOAL := bin
