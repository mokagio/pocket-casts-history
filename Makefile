BIN := pocket-casts-history

.PHONY: test run build clean

test:
	go test ./...

build: test clean
	go build -o $(BIN) .

run: build
	./$(BIN)

clean:
	rm -f $(BIN)
