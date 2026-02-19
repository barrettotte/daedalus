PREFIX ?= $(HOME)/.local
BINDIR = $(PREFIX)/bin
DATADIR = $(PREFIX)/share/daedalus
APPDIR = $(PREFIX)/share/applications

.PHONY: \
	dev build cli test lint fmt \
	clean install uninstall \
	frontend-install frontend-dev frontend-build frontend-check

dev:
	wails dev

build:
	wails build

cli:
	go build -o build/bin/daedalus-cli ./cmd/daedalus-cli

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

install:
	mkdir -p $(BINDIR) $(DATADIR) $(APPDIR)
	cp build/bin/daedalus $(BINDIR)/daedalus
	-cp build/bin/daedalus-cli $(BINDIR)/daedalus-cli
	cp build/linux/daedalus.svg $(DATADIR)/daedalus.svg
	sed 's|ICON_PATH|$(DATADIR)/daedalus.svg|' build/linux/daedalus.desktop > $(APPDIR)/daedalus.desktop
    # for GNOME
	-update-desktop-database $(APPDIR) 2>/dev/null
    # for KDE
	-kbuildsycoca6 2>/dev/null

uninstall:
	rm -f $(BINDIR)/daedalus $(BINDIR)/daedalus-cli
	rm -rf $(DATADIR)
	rm -f $(APPDIR)/daedalus.desktop
    # for GNOME
	-update-desktop-database $(APPDIR) 2>/dev/null
    # for KDE
	-kbuildsycoca6 2>/dev/null

# FRONTEND

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

frontend-check:
	cd frontend && npx svelte-check
