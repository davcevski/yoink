# yoink task runner — https://github.com/casey/just
# Run `just` to list recipes.

set shell := ["bash", "-uc"]

# show available recipes
default:
    @just --list

# run the test suite with the race detector
test:
    CGO_ENABLED=1 go test -race ./...

# format check + vet
check:
    test -z "$(gofmt -l . | grep -v '^vendor/')" || { echo "gofmt needed"; exit 1; }
    go vet ./...

# build the binary into ./yoink
build:
    CGO_ENABLED=1 go build -o yoink .

# install to $GOBIN / ~/go/bin and (re)start the local daemon
install: build
    go install .
    yoink install

# dry-run the full release locally (needs goreleaser; nothing is published)
snapshot:
    goreleaser release --snapshot --clean --skip=publish

# tag and push a release, e.g. `just release v0.1.0`
# CI then builds the universal binary and updates the Homebrew cask.
release version:
    @git diff --quiet || { echo "working tree is dirty — commit first"; exit 1; }
    @[[ "{{version}}" == v* ]] || { echo "version must start with 'v', e.g. v0.1.0"; exit 1; }
    git push origin main
    git tag -a "{{version}}" -m "{{version}}"
    git push origin "{{version}}"
