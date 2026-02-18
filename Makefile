BIN := pocket-casts-history

.PHONY: test run build clean

test:
	go test ./...

build:
	go build -o $(BIN) .

run: build
	./$(BIN)

clean:
	rm -f $(BIN)
