.PHONY: build run test vet fmt

build:
	go build -o bin/triage .

run: build
	./bin/triage -mode all -repos "$(REPOS)"

test:
	go test ./... -count=1 -v

vet:
	go vet ./...

fmt:
	gofmt -w .