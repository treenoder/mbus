PACKAGE := ./mbus/...
COVERAGE_FILE := coverage.out

.PHONY: all test coverage cover clean

all: test

test:
	go test -v $(PACKAGE) -race

coverage: clean
	go test -coverprofile=$(COVERAGE_FILE) $(PACKAGE) -race

cover: coverage
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	open coverage.html || xdg-open coverage.html || start coverage.html
	sleep 1
	$(MAKE) clean

clean:
	rm -f $(COVERAGE_FILE) coverage.html