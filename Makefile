gen:
	go generate ./...

clean_artifacts:
	cd ./artifacts && rm *

build_molt_cli:
	mkdir -p ./artifacts
	if test "$(version)" = "" ; then \
        echo "tag is not set, try running this command with a tag like 'make build_cli version=1.0.0'"; \
        exit 1; \
    fi
	./scripts/build-cross-platform.sh ./ ./artifacts/molt $(version)

sync_hooks:
	cp -a .githooks/ .git/hooks/

rewrite_fetch_tests:
	go test -v -run ^TestDataDriven github.com/cockroachdb/molt/fetch --rewrite

run_e2e_tests:
	go test -timeout 100s -run TestDataDriven github.com/cockroachdb/molt/e2e -e2e-enabled

rewrite_e2e_tests:
	go test -timeout 600s -run TestDataDriven github.com/cockroachdb/molt/e2e -e2e-enabled --rewrite