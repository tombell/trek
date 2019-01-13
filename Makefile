MODFLAGS=-mod=vendor
TESTFLAGS=-cover -v

all: test

test:
	go test ${MODFLAGS} ${TESTFLAGS} ./...

.PHONY: all test
