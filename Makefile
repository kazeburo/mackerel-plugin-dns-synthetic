VERSION=0.0.3
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION}"
all: mackerel-plugin-dns-synthetic

.PHONY: mackerel-plugin-dns-synthetic

mackerel-plugin-dns-synthetic: cmd/mackerel-plugin-dns-synthetic/main.go
	go build $(LDFLAGS) -o mackerel-plugin-dns-synthetic cmd/mackerel-plugin-dns-synthetic/main.go

linux: cmd/mackerel-plugin-dns-synthetic/main.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o mackerel-plugin-dns-synthetic cmd/mackerel-plugin-dns-synthetic/main.go

fmt:
	go fmt ./...

check:
	go test ./...

clean:
	rm -rf mackerel-plugin-dns-synthetic

tag:
	git tag v${VERSION}
	git push origin v${VERSION}
	git push origin main
