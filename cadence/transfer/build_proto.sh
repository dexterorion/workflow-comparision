#!/usr/bin/env bash

echo building protobuf stuff...

set -eo pipefail

rm -f cadence/transfer/common/protogen/*.pb.go

protoc --go_out=plugins=grpc:./cadence/transfer/common/protogen -I ./cadence/transfer/common/protodefs/  ./cadence/transfer/common/protodefs/*.proto

# go get google.golang.org/grpc


