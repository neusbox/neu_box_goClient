VERSION := $(shell tr -d '[:space:]' < VERSION)
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath -buildvcs=false -tags=netgo,osusergo

.PHONY: build test vet clean

build:
	CGO_ENABLED=0 GOOS=linux GOTOOLCHAIN=local go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o neu-sbox .
	sha256sum neu-sbox > neu-sbox.sha256

test:
	go test -count=1 ./...

vet:
	go vet ./...

clean:
	rm -f neu-sbox neu-sbox.sha256
