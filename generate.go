//go:build devcycle_wasm_bucketing

//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
//go:generate go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@latest
//go:generate protoc --go_out=. --plugin protoc-gen-go=${GOPATH}/bin/protoc-gen-go --go-vtproto_out=. --plugin protoc-gen-go-vtproto=${GOPATH}/bin/protoc-gen-go-vtproto --go-vtproto_opt=features=marshal+unmarshal+size ./proto/variableForUserParams.proto

package devcycle
