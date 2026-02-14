PHONY: all

serve:
	cd backend && go run cmd/server/main.go

all: backend

backend: assets bin

assets: backend/internal/assets/dist

backend/internal/assets/dist: 
	cd backend/internal/assets && go generate ./...

bin: server.exe

server.exe: backend/internal/assets/dist
	cd backend && go build -o server.exe cmd/server/main.go

clean:
	rm -rf backend/internal/assets/dist
	rm -f backend/server.exe