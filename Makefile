run:
	cd cmd/server && go run main.go

install:
	cd backend && pip install -r requirements.txt

upgrade:
	cd backend && alembic upgrade head

autogenerate:
	cd backend && alembic revision --autogenerate -m "migration"

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