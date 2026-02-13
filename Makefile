createdb:
	docker exec -it user-auth-service-postgres-1 createdb --username=root --owner=root auth_db

dropdb:
	docker exec -it user-auth-service-postgres-1 dropdb --username=root auth_db

run-app:
	go run main.go

compose-up:
	docker compose up -d

compose-down:
	docker compose down

proto:
	protoc -I="%PROTO_INCLUDE%" --proto_path=pkg/proto --go_out=pkg/pb --go_opt=paths=source_relative --go-grpc_out=pkg/pb --go-grpc_opt=paths=source_relative pkg/proto/*.proto

http:
	go run cmd/http/main.go

grpc:
	go run cmd/grpc/main.go

.PHONY: createdb dropdb run-app compose-up compose-down proto http grpc
