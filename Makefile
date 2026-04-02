DB_SOURCE ?= "postgresql://myuser:mypassword@localhost:5432/simple_bank?sslmode=disable"

postgres:
	docker run --name pg -e POSTGRES_USER=myuser -e POSTGRES_PASSWORD=mypassword -p 5432:5432 -d postgres:latest

createdb:
	docker exec -it pg createdb -U myuser simple_bank

dropdb:
	docker exec -it pg dropdb -U myuser simple_bank

migrateup:
	migrate -path db/migrations -database $(DB_SOURCE) -verbose up

migrateup1:
	migrate -path db/migrations -database $(DB_SOURCE) -verbose up 1

migratedown:
	migrate -path db/migrations -database $(DB_SOURCE) -verbose down

migratedown1:
	migrate -path db/migrations -database $(DB_SOURCE) -verbose down 1

migratedownall:
	migrate -path db/migrations -database $(DB_SOURCE) -verbose drop

sqlc:
	docker run --rm -v $(PWD):/src -w /src sqlc/sqlc generate

test:
	go test -v -cover ./...

server:
	go run ./...

mock:
	mockgen -package=mockdb -destination ./db/mock/store.go ./db/sqlc Store

.PHONY: postgres createdb dropdb migrateup migratedown migratedownall sqlc test server mocks