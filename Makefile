TAG := $(shell git describe --tags --dirty)
LDFLAGS = -X main.version=$(TAG)
SRCS = $(wildcard api/*.go *.go) go.mod go.sum
BINS = wsync-linux-amd64 \
       wsync-linux-arm64 \
       wsync-macos-amd64 \
       wsync-macos-arm64 \
       wsync-windows-amd64.exe

wsync: $(SRCS)
	go build -ldflags="$(LDFLAGS)"

all: wsync $(BINS)

wsync-linux-amd64: GOOS = linux
wsync-linux-amd64: GOARCH = amd64

wsync-linux-arm64: GOOS = linux
wsync-linux-arm64: GOARCH = arm64

wsync-macos-amd64: GOOS = darwin
wsync-macos-amd64: GOARCH = amd64

wsync-macos-arm64: GOOS = darwin
wsync-macos-arm64: GOARCH = arm64

wsync-windows-amd64.exe: GOOS = windows
wsync-windows-amd64.exe: GOARCH = amd64

$(BINS):
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="$(LDFLAGS)" -o $@

clean:
	rm -f wsync $(BINS)

# Create a new release
bump = echo '$2' | awk 'BEGIN{FS=OFS="."} {$$$1+=1; for (i=$1+1; i<=3; i++) $$i=0} 1'
release-patch: V := 3
release-minor: V := 2
release-major: V := 1
release-%: PREVTAG = $(shell git describe --tags --abbrev=0)
release-%: TAG = v$(shell $(call bump,$V,$(PREVTAG:v%=%)))
release-%: CONFIRM_MSG = Create release $(TAG)
release-patch release-minor release-major: release-%: .confirm clean all
	git tag $(TAG)
	git push --tags
	gh release create $(TAG)
	gh release upload $(TAG) wsync-*

.confirm:
	@echo -n "$(CONFIRM_MSG)? [y/N] " && read ans && [ $${ans:-N} = y ]

.PHONY: all clean release-patch release-minor release-major .confirm
