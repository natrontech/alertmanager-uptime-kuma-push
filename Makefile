NAME        ?= alertmanager-uptime-kuma-push
BUILD_DATE  ?= $(shell date -Iseconds)
VERSION     ?= $(shell git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD)-local
# Adds "-dirty" suffix if there are uncommitted changes in the git repository
COMMIT_REF  ?= $(shell git describe --dirty --always)

#########
# Go    #
#########

.PHONY: go-tidy
go-tidy:
	go mod tidy -compat=1.22
	@echo "Go modules tidied."

.PHONY: go-build
go-build:
	go build -o $(NAME) -trimpath -tags="netgo" -ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT_REF) -X main.BuildTime=$(BUILD_DATE)" ./cmd/pusher
	@echo "Go build completed."
