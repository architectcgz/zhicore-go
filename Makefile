.PHONY: check test test-size structure

check: structure test-size test

structure:
	bash scripts/check-structure.sh

test-size:
	python3 scripts/check-test-size.py --root .

test:
	@set -e; \
	for mod in $$(find services libs -mindepth 2 -maxdepth 2 -name go.mod -printf '%h\n' | sort); do \
		echo "go test ./... ($$mod)"; \
		(cd "$$mod" && go test ./...); \
	done
