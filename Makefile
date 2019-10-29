PKG = github.com/vt6/vt6-website-build
BIN = $(notdir $(PKG))
PREFIX = /usr

all: $(BIN)

# NOTE: This repo uses Go modules, and uses a synthetic GOPATH at
# $(CURDIR)/.gopath that is only used for the build cache. $GOPATH/src/ is
# empty.
GO            = GOPATH=$(CURDIR)/.gopath GOBIN=$(CURDIR) go
GO_BUILDFLAGS =
GO_LDFLAGS    = -s -w

$(BIN): FORCE
	$(GO) install $(GO_BUILDFLAGS) -ldflags '$(GO_LDFLAGS)' '$(PKG)'

run: $(BIN) FORCE
	./$(BIN) ../vt6/ output/

install: FORCE all
	install -D -m 0755 $(BIN) "$(DESTDIR)$(PREFIX)/bin/$(BIN)"

vendor: FORCE
	$(GO) mod tidy
	$(GO) mod vendor

.PHONY: FORCE
