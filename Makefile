.PHONY: dev build test clean install frontend-dev frontend-build

dev:
	wails dev

build:
	wails build

test:
	go test -v ./...

install:
	cd frontend && npm install

clean:
	rm -rf build/bin
	rm -rf frontend/dist

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build
