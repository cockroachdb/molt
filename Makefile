gen_molt_service:
	go run goa.design/goa/v3/cmd/goa gen github.com/cockroachdb/molt/moltservice/design -o ./moltservice

gen_ts_api:
	cd ui && pnpm openapi --input ../moltservice/gen/http/openapi3.yaml --output ./src/apigen/

gen:
	@$(MAKE) gen_molt_service
	@$(MAKE) gen_ts_api
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

run_molt_svc:
	cd moltservice && MOLT_SERVICE_ALLOW_ORIGIN="/.*localhost.*/" go run .