.PHONY: test test-unit test-integration test-coverage test-e2e test-all

# Run all backend tests (unit + integration)
test:
	AGENTSTORE_ENV=test go test -count=1 -v ./...

# Run only unit tests (no DB required)
test-unit:
	AGENTSTORE_ENV=test go test -short -count=1 ./...

# Run only integration tests (require test DB)
test-integration:
	AGENTSTORE_ENV=test go test -count=1 -v -run Integration ./...

# Generate coverage report
test-coverage:
	AGENTSTORE_ENV=test go test -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run Playwright E2E tests (requires running frontend)
test-e2e:
	cd ../frontend && npx playwright test

# Run everything
test-all: test test-e2e
