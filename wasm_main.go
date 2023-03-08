package devcycle

import (
	_ "embed"
	"github.com/bytecodealliance/wasmtime-go/v6"
)

//go:embed bucketing-lib.release.wasm
var wasmMainBinary []byte

type WASMMain struct {
	wasm       []byte
	wasmEngine *wasmtime.Engine
	wasmModule *wasmtime.Module
}

func (d *WASMMain) Initialize() (err error) {
	d.wasm = wasmMainBinary
	d.wasmEngine = wasmtime.NewEngine()
	d.wasmModule, err = wasmtime.NewModule(d.wasmEngine, wasmMainBinary)

	if err != nil {
		return
	}

	return
}
