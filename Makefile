PKG = github.com/vt6/vt6-website-build
BIN = $(basename $(PKG))
PREFIX = /usr

all: $(BIN)

GO            = GOPATH=$(CURDIR)/.gopath GOBIN=$(CURDIR) go
GO_BUILDFLAGS =
GO_LDFLAGS    = -s -w

$(BIN): FORCE
	$(GO) install $(GO_BUILDFLAGS) -ldflags '$(GO_LDFLAGS)' '$(PKG)'

install: FORCE all
	install -D -m 0755 build/$(BIN) "$(DESTDIR)$(PREFIX)/bin/$(BIN)"

# vendoring by https://github.com/holocm/golangvend
vendor: FORCE
	@golangvend

.PHONY: FORCE
