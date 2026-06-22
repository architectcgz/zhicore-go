.PHONY: check test structure

check: structure test

structure:
	bash scripts/check-structure.sh

test:
	@set -e; \
	for mod in $$(find services libs -mindepth 2 -maxdepth 2 -name go.mod -printf '%h\n' | sort); do \
		echo "go test ./... ($$mod)"; \
		(cd "$$mod" && go test ./...); \
	done
