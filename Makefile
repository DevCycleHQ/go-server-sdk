GOLANGCI_VERSION=v1.52.2
BINDIR=$(shell pwd)/.bin
TAGS_PARAM=$(if ${TAGS},-tags ${TAGS},)
RACE_PARAM=$(if ${RACE},-race,)

${BINDIR}:
	@mkdir -p ${BINDIR}

${BINDIR}/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${BINDIR} ${GOLANGCI_VERSION}

lint: ${BINDIR} ${BINDIR}/golangci-lint
	golangci-lint run --sort-results --skip-files proto --disable unused && \
	golangci-lint run --sort-results --skip-files proto --build-tags devcycle_wasm_bucketing --disable unused
	
test:
	go test -v ${RACE_PARAM} ${TAGS_PARAM} ./...

.PHONY: lint
