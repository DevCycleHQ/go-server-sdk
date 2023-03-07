package devcycle

import (
	_ "embed"
	"github.com/bytecodealliance/wasmtime-go/v6"
)

//go:embed bucketing-lib.release.wasm
var wasmMainBinary []byte

type WASMMain struct {
	wasm       []byte
	wasmLinker *wasmtime.Linker
	wasmEngine *wasmtime.Engine
	wasmModule *wasmtime.Module
}

func (d *WASMMain) Initialize(options *DVCOptions) (err error) {
	d.wasm = wasmMainBinary
	d.wasmEngine = wasmtime.NewEngine()
	d.wasmLinker = wasmtime.NewLinker(d.wasmEngine)
	err = d.wasmLinker.DefineWasi()

	if err != nil {
		return
	}

	d.wasmModule, err = wasmtime.NewModule(d.wasmEngine, d.wasm)

	return
}

func (d *WASMMain) GetWasmLinker() *wasmtime.Linker {
	return d.wasmLinker
}
