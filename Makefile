.PHONY: build clean

build: clean
	env GOOS=darwin go build -ldflags="-s -w" -o bin/darwin/awssecgroup src/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/linux/awssecgroup src/main.go

clean:
	rm -rf ./bin