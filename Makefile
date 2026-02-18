BINARY := scribe2srt
VERSION ?= dev
LDFLAGS := -s -w -X scribe2srt/cmd.Version=$(VERSION)

.PHONY: build clean run test

build:
	go build -ldflags '$(LDFLAGS)' -o $(BINARY) .

clean:
	rm -f $(BINARY)

run:
	go run -ldflags '$(LDFLAGS)' . $(ARGS)

test:
	go test ./...
