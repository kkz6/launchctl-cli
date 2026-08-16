.PHONY: build run clean install uninstall release-dry release-patch release-minor release-major

CURRENT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
VERSION := $(shell echo $(CURRENT_TAG) | sed 's/^v//')
BINARY = bin/lctl
LDFLAGS = -ldflags "-X github.com/kkz6/launchctl/cmd.Version=$(VERSION)"
PREFIX ?= /usr/local

build:
	go build $(LDFLAGS) -o $(BINARY) .

run: build
	./$(BINARY)

install: build
	install -d $(PREFIX)/bin
	install -m 755 $(BINARY) $(PREFIX)/bin/lctl

uninstall:
	rm -f $(PREFIX)/bin/lctl

clean:
	rm -rf bin/ dist/

# Dry run goreleaser locally (no publish)
release-dry:
	goreleaser release --snapshot --clean

# Tag and push — triggers GitHub Actions → goreleaser → brew tap update
release-patch:
	@$(MAKE) _release BUMP=patch

release-minor:
	@$(MAKE) _release BUMP=minor

release-major:
	@$(MAKE) _release BUMP=major

_release:
	@CURRENT=$(CURRENT_TAG); \
	MAJOR=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f1); \
	MINOR=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f2); \
	PATCH=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f3); \
	case $(BUMP) in \
		major) MAJOR=$$((MAJOR + 1)); MINOR=0; PATCH=0 ;; \
		minor) MINOR=$$((MINOR + 1)); PATCH=0 ;; \
		patch) PATCH=$$((PATCH + 1)) ;; \
	esac; \
	NEXT="v$$MAJOR.$$MINOR.$$PATCH"; \
	PLUGIN_VERSION=$$(python3 -c 'import json; print(json.load(open("plugins/launchctl/.codex-plugin/plugin.json"))["version"])'); \
	if [ "$$PLUGIN_VERSION" != "$${NEXT#v}" ]; then \
		echo "Plugin version $$PLUGIN_VERSION must match $$NEXT before release."; \
		exit 1; \
	fi; \
	echo "$(CURRENT_TAG) → $$NEXT"; \
	echo ""; \
	git log --oneline $(CURRENT_TAG)..HEAD 2>/dev/null || echo "  (first release)"; \
	echo ""; \
	read -p "Tag $$NEXT and push? [y/N] " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		git tag -a $$NEXT -m "Release $$NEXT"; \
		git push origin $$NEXT; \
		echo ""; \
		echo "Tagged and pushed $$NEXT — GitHub Actions will handle the rest."; \
	else \
		echo "Aborted."; \
	fi
