SRCS = $(wildcard api/*.go *.go) go.mod go.sum
BINS = wsync-linux-amd64 \
       wsync-linux-arm64 \
       wsync-macos-amd64 \
       wsync-macos-arm64 \
       wsync-windows-amd64.exe

wsync: $(SRCS)
	go build

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
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@

clean:
	rm -f wsync $(BINS)

.PHONY: all clean
