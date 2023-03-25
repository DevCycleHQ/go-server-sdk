package devcycle

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/tetratelabs/wazero"
)

//go:embed bucketing-lib.release.wasm
var wasmMainBinary []byte

//go:embed bucketing-lib.debug.wasm
var wasmDebugBinary []byte

type WASMMain struct {
	wasm          []byte
	wazeroRuntime wazero.Runtime
	wazeroModule  wazero.CompiledModule
}

func (d *WASMMain) Initialize(options *DVCOptions) (err error) {
	d.wasm = wasmMainBinary
	if options != nil && options.UseDebugWASM {
		infof("Using debug WASM binary. (This is not recommended for production use)")
		d.wasm = wasmDebugBinary
	}
	d.wazeroRuntime = wazero.NewRuntimeWithConfig(context.Background(), wazero.NewRuntimeConfigCompiler())
	d.wazeroModule, err = d.wazeroRuntime.CompileModule(context.Background(), d.wasm)
	if err != nil {
		return fmt.Errorf("failed to compile wasm module: %w", err)
	}

	return nil
}
