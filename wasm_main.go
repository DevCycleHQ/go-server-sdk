package devcycle

import (
	_ "embed"
	"github.com/bytecodealliance/wasmtime-go/v6"
	"github.com/devcyclehq/go-server-sdk/v2/util"
)

//go:embed bucketing-lib.release.wasm
var wasmMainBinary []byte

//go:embed bucketing-lib.debug.wasm
var wasmDebugBinary []byte

type WASMMain struct {
	wasm       []byte
	wasmEngine *wasmtime.Engine
	wasmModule *wasmtime.Module
}

func (d *WASMMain) Initialize(options *DVCOptions) (err error) {
	d.wasm = wasmMainBinary
	d.wasmEngine = wasmtime.NewEngine()
	if options != nil && options.UseDebugWASM {
		util.Infof("Using debug WASM binary. (This is not recommended for production use)")
		d.wasm = wasmDebugBinary
	}
	d.wasmModule, err = wasmtime.NewModule(d.wasmEngine, d.wasm)

	if err != nil {
		return
	}

	return
}
