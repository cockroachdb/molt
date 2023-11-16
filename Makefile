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