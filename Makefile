BINARY := scribe2srt
LDFLAGS := -s -w

.PHONY: build clean run test

build:
	go build -ldflags '$(LDFLAGS)' -o $(BINARY) .

clean:
	rm -f $(BINARY)

run:
	go run -ldflags '$(LDFLAGS)' . $(ARGS)

test:
	go test ./...
