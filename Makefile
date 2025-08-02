createdb:
	docker exec -it user-auth-service-postgres-1 createdb --username=root --owner=root auth_db

dropdb:
	docker exec -it user-auth-service-postgres-1 dropdb --username=root auth_db

run-app:
	go run main.go

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

.PHONY: createdb dropdb run-app compose-up compose-down
