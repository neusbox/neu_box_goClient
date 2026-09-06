VERSION := $(shell tr -d '[:space:]' < VERSION)
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath -buildvcs=false -tags=netgo,osusergo
GOCMD ?= go

# 本项目需要 Go >= 1.18（any / strings.Cut / -buildvcs）；老工具链给出明确报错。
GOVERSION := $(shell $(GOCMD) version 2>/dev/null | sed -n 's/.*go\([0-9][0-9]*\)\.\([0-9][0-9]*\).*/\1 \2/p')
GOFULLVERSION := $(shell $(GOCMD) version 2>/dev/null | head -n 1)

.PHONY: build test vet clean check-go

check-go:
	@if [ -z "$(GOVERSION)" ]; then \
		echo "error: 无法检测 Go 版本（未安装 $(GOCMD)?）" >&2; exit 1; \
	fi; \
	major="$$(echo "$(GOVERSION)" | cut -d' ' -f1)"; \
	minor="$$(echo "$(GOVERSION)" | cut -d' ' -f2)"; \
	if [ "$$major" -gt 1 ] || { [ "$$major" -eq 1 ] && [ "$$minor" -ge 18 ]; }; then \
		:; \
	else \
		echo "error: 需要 Go >= 1.18，当前: $(GOFULLVERSION)" >&2; exit 1; \
	fi

build: check-go
	CGO_ENABLED=0 GOOS=linux GOTOOLCHAIN=local $(GOCMD) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o neu-sbox .
	sha256sum neu-sbox > neu-sbox.sha256

test:
	go test -count=1 ./...

vet:
	go vet ./...

clean:
	rm -f neu-sbox neu-sbox.sha256
