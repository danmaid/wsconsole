.PHONY: build clean install test test-docker run systemd-build systemd-up systemd-down systemd-shell systemd-start systemd-stop systemd-logs dev test-launchers test-direct test-systemd-run test-down test-client test-logs test-direct-logs test-systemd-run-logs build-linux build-linux-amd64 build-linux-arm64 build-linux-armv7 build-deb deb deb-amd64 deb-arm64 deb-armhf deb-test deb-test-build deb-test-up deb-test-down deb-test-shell deb-test-logs deb-test-status deb-test-verify setup-users

BINARY_NAME=wsconsole
INSTALL_PATH=/usr/local/bin
STATIC_PATH=/usr/local/share/wsconsole/static
VERSION ?= 1.0.0
DEB_ARCH ?= amd64
DEB_GOARCH ?= amd64
DEB_GOARM ?=
SYSTEMD_COMPOSE_FILE=docker-compose.systemd.yml
TEST_COMPOSE_FILE=docker-compose.test.yml
DEB_TEST_COMPOSE_FILE=docker-compose.deb-test.yml
SYSTEMD_SERVICE=systemd-test
WSCONSOLE_URL=http://localhost:8080
DEB_TEST_URL=http://localhost:8083

build:
	go build -o $(BINARY_NAME) ./cmd/wsconsole

build-linux: build-linux-amd64

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/wsconsole

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/wsconsole

build-linux-armv7:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/wsconsole

build-deb:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(DEB_GOARCH) GOARM=$(DEB_GOARM) go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/wsconsole

clean:
	rm -f $(BINARY_NAME)
	rm -f wsconsole_*.deb
	rm -rf packaging/deb/build/

test:
	go test -v ./...

test-docker:
	docker run --rm -v "$(PWD)":/src -w /src golang:1.21 go test -v ./...

run:
	go run ./cmd/wsconsole

systemd-build:
	docker compose -f $(SYSTEMD_COMPOSE_FILE) build

systemd-up:
	docker compose -f $(SYSTEMD_COMPOSE_FILE) up -d --build

systemd-down:
	docker compose -f $(SYSTEMD_COMPOSE_FILE) down

systemd-shell:
	docker compose -f $(SYSTEMD_COMPOSE_FILE) exec systemd-test bash

systemd-start: systemd-up
	@echo "Waiting for systemd to start..."
	@sleep 5
	docker compose -f $(SYSTEMD_COMPOSE_FILE) exec systemd-test bash -c "systemctl enable wsconsole && systemctl start wsconsole"
	@echo ""
	@echo "✅ wsconsole is running!"
	@echo "🌐 Access URL: $(WSCONSOLE_URL)"
	@echo ""

systemd-stop: systemd-down
	@echo "✅ wsconsole stopped"

systemd-logs:
	docker compose -f $(SYSTEMD_COMPOSE_FILE) exec systemd-test journalctl -u wsconsole -f

dev: build
	@echo "🚀 Starting wsconsole in development mode..."
	@echo "   Strategy: direct (UID=$(shell id -u))"
	@echo "   Port: 8080"
	@echo "   Log level: debug"
	@echo ""
	@echo "🌐 Access URLs:"
	@echo "   WebSocket: ws://localhost:8080/ws"
	@echo "   Web UI:    http://localhost:8080/"
	@echo "   Health:    http://localhost:8080/healthz"
	@echo ""
	@echo "🔐 Login credentials (run 'make setup-users' if needed):"
	@echo "   testuser / testpass"
	@echo "   vscode   / vscode"
	@echo "   root     / root"
	@echo ""
	@echo "📋 Press Ctrl+C to stop"
	@echo ""
	@trap 'echo ""; echo "⛔ Stopped wsconsole"; exit 0' INT; \
	./$(BINARY_NAME) -addr :8080 -launcher direct -log debug -static ./deploy/static

setup-users:
	@echo "👤 Setting up test users..."
	@if ! id testuser >/dev/null 2>&1; then \
		useradd -m -s /bin/bash testuser && echo "testuser:testpass" | chpasswd && \
		echo "✓ Created testuser / testpass"; \
	else \
		echo "testuser:testpass" | chpasswd && \
		echo "✓ Updated testuser password"; \
	fi
	@echo "vscode:vscode" | chpasswd 2>/dev/null && echo "✓ Set vscode password" || true
	@echo "root:root" | chpasswd && echo "✓ Set root password"
	@echo ""
	@echo "Available users:"
	@echo "  - testuser / testpass"
	@echo "  - vscode   / vscode"
	@echo "  - root     / root"
	@echo ""
	@echo "⚠️  Note: Dev container has custom prompt. For standard prompt (#/$$),"
	@echo "    use test containers: make test-launchers"

install: build-linux
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	sudo mkdir -p $(STATIC_PATH)
	sudo cp deploy/static/index.html $(STATIC_PATH)/
	sudo cp deploy/systemd/wsconsole.service /etc/systemd/system/
	sudo cp deploy/polkit/10-wsconsole.rules /etc/polkit-1/rules.d/
	sudo systemctl daemon-reload

deb: build-deb
	mkdir -p packaging/deb/build/usr/local/bin
	mkdir -p packaging/deb/build/usr/local/share/wsconsole/static
	mkdir -p packaging/deb/build/etc/systemd/system
	mkdir -p packaging/deb/build/etc/polkit-1/rules.d
	mkdir -p packaging/deb/build/DEBIAN
	cp $(BINARY_NAME) packaging/deb/build/usr/local/bin/
	cp deploy/static/index.html packaging/deb/build/usr/local/share/wsconsole/static/
	cp deploy/systemd/wsconsole.service packaging/deb/build/etc/systemd/system/
	cp deploy/polkit/10-wsconsole.rules packaging/deb/build/etc/polkit-1/rules.d/
	cp packaging/deb/DEBIAN/control packaging/deb/build/DEBIAN/
	cp packaging/deb/DEBIAN/postinst packaging/deb/build/DEBIAN/
	cp packaging/deb/DEBIAN/prerm packaging/deb/build/DEBIAN/
	sed -i "s/^Version:.*/Version: $(VERSION)/" packaging/deb/build/DEBIAN/control
	sed -i "s/^Architecture:.*/Architecture: $(DEB_ARCH)/" packaging/deb/build/DEBIAN/control
	chmod 755 packaging/deb/build/DEBIAN/postinst
	chmod 755 packaging/deb/build/DEBIAN/prerm
	chmod 755 packaging/deb/build/usr/local/bin/$(BINARY_NAME)
	dpkg-deb --build packaging/deb/build wsconsole_$(VERSION)_$(DEB_ARCH).deb

deb-amd64:
	@$(MAKE) deb VERSION=$(VERSION) DEB_ARCH=amd64 DEB_GOARCH=amd64

deb-arm64:
	@$(MAKE) deb VERSION=$(VERSION) DEB_ARCH=arm64 DEB_GOARCH=arm64

deb-armhf:
	@$(MAKE) deb VERSION=$(VERSION) DEB_ARCH=armhf DEB_GOARCH=arm DEB_GOARM=7

# Test launcher strategies
test-launchers: build
	@echo "================================================"
	@echo "Building wsconsole binary..."
	@echo "================================================"
	@echo "✓ Build complete"
	@echo ""
	@echo "================================================"
	@echo "Starting test containers..."
	@echo "================================================"
	docker compose -f $(TEST_COMPOSE_FILE) up -d --build
	@echo ""
	@echo "================================================"
	@echo "Test Environment Ready!"
	@echo "================================================"
	@echo ""
	@echo "Test 1: Direct Launcher (UID=0)"
	@echo "  URL: http://localhost:8081/ws"
	@echo "  Strategy: direct"
	@echo "  Description: Runs as root, directly forks /bin/login"
	@echo "  Logs: docker compose -f $(TEST_COMPOSE_FILE) logs -f test-direct"
	@echo ""
	@echo "Test 2: SystemdRun Launcher"
	@echo "  URL: http://localhost:8082/ws"
	@echo "  Strategy: systemd-run"
	@echo "  Description: Uses systemd-run for privilege escalation"
	@echo "  Logs: docker compose -f $(TEST_COMPOSE_FILE) logs -f test-systemd-run"
	@echo ""
	@echo "================================================"
	@echo "Test Credentials:"
	@echo "  Username: testuser"
	@echo "  Password: testpass"
	@echo "================================================"
	@echo ""
	@echo "To stop: docker compose -f $(TEST_COMPOSE_FILE) down"
	@echo "To view all logs: docker compose -f $(TEST_COMPOSE_FILE) logs -f"
	@echo ""

test-direct: build
	@echo "Starting Direct Launcher test container (UID=0)..."
	docker compose -f $(TEST_COMPOSE_FILE) up -d --build test-direct
	@echo ""
	@echo "✅ Direct Launcher test running on port 8081"
	@echo "🌐 WebSocket URL: ws://localhost:8081/ws"
	@echo "📋 View logs: make test-direct-logs"
	@echo ""

test-systemd-run: build
	@echo "Starting SystemdRun Launcher test container..."
	docker compose -f $(TEST_COMPOSE_FILE) up -d --build test-systemd-run
	@echo ""
	@echo "✅ SystemdRun Launcher test running on port 8082"
	@echo "🌐 WebSocket URL: ws://localhost:8082/ws"
	@echo "📋 View logs: make test-systemd-run-logs"
	@echo ""

test-down:
	@echo "Stopping test containers..."
	docker compose -f $(TEST_COMPOSE_FILE) down
	@echo "✅ Test containers stopped"

test-direct-logs:
	docker compose -f $(TEST_COMPOSE_FILE) logs -f test-direct

test-systemd-run-logs:
	docker compose -f $(TEST_COMPOSE_FILE) logs -f test-systemd-run

test-client:
	@LAUNCHER=$${LAUNCHER:-direct}; \
	if [ "$$LAUNCHER" = "systemd-run" ]; then PORT=8082; else PORT=8081; fi; \
	URL="ws://localhost:$$PORT/ws?launcher=$$LAUNCHER"; \
	echo "Testing launcher: $$LAUNCHER"; \
	echo "Connecting to: $$URL"; \
	echo "Credentials: testuser / testpass"; \
	echo "---"; \
	if command -v websocat >/dev/null 2>&1; then \
		echo "Using websocat..."; \
		websocat "$$URL"; \
	elif command -v wscat >/dev/null 2>&1; then \
		echo "Using wscat..."; \
		wscat -c "$$URL"; \
	else \
		echo "Error: Please install websocat or wscat"; \
		echo ""; \
		echo "Install websocat:"; \
		echo "  cargo install websocat"; \
		echo ""; \
		echo "Install wscat:"; \
		echo "  npm install -g wscat"; \
		exit 1; \
	fi


# Debian package testing targets
deb-test: deb-test-build deb-test-up deb-test-status
	@echo ""
	@echo "✅ Debian package test environment is ready!"
	@echo ""
	@echo "🌐 Access URLs:"
	@echo "   Web UI:    $(DEB_TEST_URL)/"
	@echo "   WebSocket: ws://localhost:8083/ws"
	@echo "   Health:    $(DEB_TEST_URL)/healthz"
	@echo ""
	@echo "📋 Useful commands:"
	@echo "   Shell:     make deb-test-shell"
	@echo "   Logs:      make deb-test-logs"
	@echo "   Status:    make deb-test-status"
	@echo "   Stop:      make deb-test-down"
	@echo ""

deb-test-build: deb
	@echo "🔨 Building deb test container..."
	docker compose -f $(DEB_TEST_COMPOSE_FILE) build

deb-test-up:
	@echo "🚀 Starting deb test container..."
	docker compose -f $(DEB_TEST_COMPOSE_FILE) up -d
	@echo "⏳ Waiting for systemd to initialize..."
	@sleep 5
	@echo "🔧 Starting wsconsole service..."
	@docker compose -f $(DEB_TEST_COMPOSE_FILE) exec -T deb-test systemctl start wsconsole.service || true
	@sleep 2

deb-test-down:
	@echo "⛔ Stopping deb test container..."
	docker compose -f $(DEB_TEST_COMPOSE_FILE) down
	@echo "✅ Container stopped"

deb-test-shell:
	@docker compose -f $(DEB_TEST_COMPOSE_FILE) exec deb-test bash

deb-test-logs:
	@echo "📋 Fetching wsconsole service logs..."
	@docker compose -f $(DEB_TEST_COMPOSE_FILE) exec deb-test journalctl -u wsconsole.service -f

deb-test-status:
	@echo "📊 Service status:"
	@docker compose -f $(DEB_TEST_COMPOSE_FILE) exec -T deb-test systemctl status wsconsole.service --no-pager || true
	@echo ""
	@echo "🔍 Port check:"
	@docker compose -f $(DEB_TEST_COMPOSE_FILE) exec -T deb-test netstat -tlnp | grep 8080 || echo "   Port 8080 not listening"

deb-test-verify:
	@echo "==================================="
	@echo "wsconsole Debian Package Test"
	@echo "==================================="
	@echo ""
	@if ! docker ps | grep -q wsconsole-deb-test; then \
		echo "✗ Container is not running"; \
		echo "Run 'make deb-test' first"; \
		exit 1; \
	fi
	@echo "✓ Container is running"
	@echo ""
	@echo "Test 1: Health check endpoint..."
	@docker exec wsconsole-deb-test curl -sf http://localhost:8080/healthz >/dev/null && echo "✓ Health check passed" || (echo "✗ Health check failed"; exit 1)
	@echo "Test 2: Web UI accessibility..."
	@docker exec wsconsole-deb-test curl -sf http://localhost:8080/ | grep -q "wsconsole" && echo "✓ Web UI accessible" || (echo "✗ Web UI not accessible"; exit 1)
	@echo "Test 3: SystemD service status..."
	@docker exec wsconsole-deb-test systemctl is-active wsconsole.service >/dev/null && echo "✓ Service is active" || (echo "✗ Service is not active"; exit 1)
	@echo "Test 4: Binary files..."
	@docker exec wsconsole-deb-test test -f /usr/local/bin/wsconsole && echo "✓ Binary installed" || (echo "✗ Binary not found"; exit 1)
	@echo "Test 5: Static files..."
	@docker exec wsconsole-deb-test test -f /usr/local/share/wsconsole/static/index.html && echo "✓ Static files installed" || (echo "✗ Static files not found"; exit 1)
	@echo "Test 6: SystemD unit file..."
	@docker exec wsconsole-deb-test test -f /etc/systemd/system/wsconsole.service && echo "✓ SystemD unit file installed" || (echo "✗ SystemD unit file not found"; exit 1)
	@echo "Test 7: Polkit rules..."
	@docker exec wsconsole-deb-test test -f /etc/polkit-1/rules.d/10-wsconsole.rules && echo "✓ Polkit rules installed" || (echo "✗ Polkit rules not found"; exit 1)
	@echo "Test 8: Launcher strategy..."
	@docker exec wsconsole-deb-test journalctl -u wsconsole.service --no-pager | grep -q "launcher_strategy.*systemd-run" && echo "✓ Using systemd-run launcher" || (echo "✗ Not using systemd-run launcher"; exit 1)
	@echo ""
	@echo "==================================="
	@echo "All tests passed!"
	@echo "==================================="
test-logs:
	docker compose -f $(TEST_COMPOSE_FILE) logs -f
