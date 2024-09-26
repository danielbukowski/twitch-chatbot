BUILD_TAGS=sqlite_userauth

migration-up:
	goose -dir ./db/migrations sqlite3 ./db/database.db up

migration-reset:
	goose -dir ./db/migrations sqlite3 ./db/database.db reset

run:
	go run -tags="${BUILD_TAGS}" cmd/main/main.go -dev

build-windows:
	go build -o ./tmp/main.exe -tags="${BUILD_TAGS}" ./cmd/main/main.go

check-build:
	go build -v -tags="${BUILD_TAGS}" ./...

build:
	go build -o=./tmp/main -tags="${BUILD_TAGS}" ./cmd/main/main.go

test:
	go test -v -race ./internal/...

lint:
	golangci-lint run ./... 

tidy:
	go mod tidy