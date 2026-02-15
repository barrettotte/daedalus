.PHONY: \
	dev build test lint fmt \
	clean \
	frontend-install frontend-dev frontend-build frontend-check

dev:
	wails dev

build:
	wails build

test:
	go test -v ./...

lint:
	gofmt -l . ./pkg/daedalus/
	go vet ./...

fmt:
	gofmt -w . ./pkg/daedalus/

clean:
	rm -rf build/bin
	rm -rf frontend/dist

# FRONTEND

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

frontend-check:
	cd frontend && npx svelte-check
