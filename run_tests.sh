#!/usr/bin/env bash
#
# prerequisites: make GITKEY=$GITKEY deps (see README.md)
#
# usage: 
#
#   sudo -E bash run_tests.sh 
#
# or
#
#   sudo -E bash run_tests.sh pattern
#
# to run tests matching "pattern" (go test -run, under the hood)

set -ex

GO_BIN=${GO_BIN:-go}
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

# Get cobertura dependencies
${GO_BIN} get github.com/t-yuki/gocover-cobertura
${GO_BIN} install github.com/t-yuki/gocover-cobertura
${GO_BIN} mod vendor

if [ -z "$1" ]
then
	find . -name '*_test.go' | uniq | sed 's|/[^/]*$|/|' | grep -v '/build/' | xargs ${GO_BIN} test -v -coverprofile=coverage.out -timeout 30m
else
	${GO_BIN} test -v -coverprofile=coverage.out -timeout 30m $@
fi 

${GO_BIN} tool cover -html=coverage.out -o coverage.html
gocover-cobertura < coverage.out > coverage.xml
