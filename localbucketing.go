package devcycle

import (
	"encoding/binary"
	"fmt"
	"github.com/bytecodealliance/wasmtime-go"
	"time"
)

type DevCycleLocalBucketing struct {
	wasm         []byte
	wasmStore    *wasmtime.Store
	wasmModule   *wasmtime.Module
	wasmInstance *wasmtime.Instance
	wasmLinker   *wasmtime.Linker
	wasiConfig   *wasmtime.WasiConfig
	wasmMemory   *wasmtime.Memory
}

func (d *DevCycleLocalBucketing) Initialize() {
	var err error

	d.wasiConfig = wasmtime.NewWasiConfig()
	d.wasiConfig.InheritEnv()
	d.wasiConfig.InheritStderr()
	d.wasiConfig.InheritStdout()
	d.wasmStore = wasmtime.NewStore(wasmtime.NewEngine())
	d.wasmStore.SetWasi(d.wasiConfig)
	d.wasmLinker = wasmtime.NewLinker(d.wasmStore.Engine)
	err = d.wasmLinker.DefineWasi()
	if err != nil {
		return
	}

	d.wasmModule, err = wasmtime.NewModule(d.wasmStore.Engine, d.wasm)
	if err != nil {
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "Date.now", func() int64 { return time.Now().UnixMilli() })
	if err != nil {
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "abort", func(messagePtr, filenamePointer, lineNum, colNum int) {
		panic(readAssemblyScriptString(messagePtr, d.wasmMemory, d.wasmStore))
	})
	if err != nil {
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "console.log", func(messagePtr int) {
		fmt.Println(readAssemblyScriptString(messagePtr, d.wasmMemory, d.wasmStore))
	})
	if err != nil {
		return
	}

	d.wasmInstance, err = d.wasmLinker.Instantiate(d.wasmStore, d.wasmModule)
	if err != nil {
		panic(err)
	}
	d.wasmMemory = d.wasmInstance.GetExport(d.wasmStore, "memory").Memory()
}

func (d *DevCycleLocalBucketing) GenerateBucketedConfigForUser(token, user string) (string, error) {
	tokenAddr, err := d.newWASMString(token)
	if err != nil {
		return "", err
	}
	userAddr, err := d.newWASMString(user)
	if err != nil {
		return "", err
	}
	_generateBucketedConfigForUser := d.wasmInstance.GetExport(d.wasmStore, "generateBucketedConfigForUser").Func()
	configPtr, err := _generateBucketedConfigForUser.Call(d.wasmStore, tokenAddr, userAddr)
	if err != nil {
		return "", err
	}
	return readAssemblyScriptString(configPtr.(int), d.wasmMemory, d.wasmStore), nil
}

func (d *DevCycleLocalBucketing) StoreConfig(config string) error {
	configAddr, err := d.newWASMString(config)
	if err != nil {
		return err
	}
	_setConfigData := d.wasmInstance.GetExport(d.wasmStore, "setConfigData").Func()
	_, err = _setConfigData.Call(d.wasmStore, configAddr)
	if err != nil {
		return err
	}
	return nil
}

func (d *DevCycleLocalBucketing) SetPlatformData(platformData string) error {
	configAddr, err := d.newWASMString(platformData)
	if err != nil {
		return err
	}
	_setPlatformData := d.wasmInstance.GetExport(d.wasmStore, "setPlatformData").Func()
	_, err = _setPlatformData.Call(d.wasmStore, configAddr)
	if err != nil {
		return err
	}
	return nil
}

func (d *DevCycleLocalBucketing) newWASMString(param string) (int, error) {
	const objectIdString = 1
	__new := d.wasmInstance.GetExport(d.wasmStore, "__new").Func()
	ptr, err := __new.Call(d.wasmStore, len(param), param)
	if err != nil {
		return -1, err
	}
	return ptr.(int), nil
}

// https://www.assemblyscript.org/runtime.html#memory-layout
func readAssemblyScriptString(pointer int, memory *wasmtime.Memory, store *wasmtime.Store) string {
	stringLength := binary.BigEndian.Uint32(memory.UnsafeData(store)[pointer-4 : pointer])
	return string(memory.UnsafeData(store)[pointer : pointer+int(stringLength)])
}
