BIN = vt6-website-build
PREFIX = /usr

all: $(BIN)

GO_BUILDFLAGS = -mod vendor
GO_LDFLAGS    = 

$(BIN): FORCE
	go build $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -o $@ .

run: $(BIN) FORCE
	./$(BIN) ../vt6/ output/

install: FORCE all
	install -D -m 0755 $(BIN) "$(DESTDIR)$(PREFIX)/bin/$(BIN)"

vendor: FORCE
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: FORCE
