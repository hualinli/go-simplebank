db_url = "postgresql://myuser:mypassword@localhost:5432/simple_bank?sslmode=disable"


postgres:
	docker run --name pg -e POSTGRES_USER=myuser -e POSTGRES_PASSWORD=mypassword -p 5432:5432 -d postgres:latest
createdb:
	docker exec -it pg createdb -U myuser simple_bank

dropdb:
	docker exec -it pg dropdb -U myuser simple_bank

migrateup:
	migrate -path db/migrations -database $(db_url) -verbose up

migratedown:
	migrate -path db/migrations -database $(db_url) -verbose down

migratedownall:
	migrate -path db/migrations -database $(db_url) -verbose drop

sqlc:
	docker run --rm -v $(PWD):/src -w /src sqlc/sqlc generate

.PHONY: postgres createdb dropdb migrateup migratedown migratedownall sqlc