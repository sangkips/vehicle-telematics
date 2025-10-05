run:
	cd cmd/server && go run main.go

ps:
	docker compose ps

up:
	docker compose up -d

stop:
	docker compose stop

rm: stop
	docker compose rm -f

test:
	go test -v ./internal/websocket ./pkg/batch ./internal/api/handlers