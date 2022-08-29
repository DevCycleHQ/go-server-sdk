package devcycle

import (
	"encoding/json"
	"fmt"
	"github.com/bytecodealliance/wasmtime-go"
	"log"
	"time"
	"unicode/utf16"
	"unsafe"
)

type DevCycleLocalBucketing struct {
	wasm          []byte
	wasmStore     *wasmtime.Store
	wasmModule    *wasmtime.Module
	wasmInstance  *wasmtime.Instance
	wasmLinker    *wasmtime.Linker
	wasiConfig    *wasmtime.WasiConfig
	wasmMemory    *wasmtime.Memory
	configManager *EnvironmentConfigManager
}

func (d *DevCycleLocalBucketing) Initialize() (err error) {

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
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "Date.now", func() float64 { return float64(time.Now().UnixMilli()) })
	if err != nil {
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "abort", func(messagePtr, filenamePointer, lineNum, colNum int32) {
		panic(readAssemblyScriptString(messagePtr, d.wasmMemory, d.wasmStore))
	})
	if err != nil {
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "console.log", func(messagePtr int32) {
		fmt.Println(readAssemblyScriptString(messagePtr, d.wasmMemory, d.wasmStore))
	})
	if err != nil {
		return
	}

	d.wasmInstance, err = d.wasmLinker.Instantiate(d.wasmStore, d.wasmModule)
	if err != nil {
		return
	}
	d.wasmMemory = d.wasmInstance.GetExport(d.wasmStore, "memory").Memory()

	log.Println("Initializing DevCycle LocalBucketing")
	d.configManager = &EnvironmentConfigManager{localBucketing: d}

	return nil
}

func (d *DevCycleLocalBucketing) InitializeConfigManager(environmentKey string, options *DVCOptions) error {
	err := d.configManager.Initialize(environmentKey, options)
	return err
}

func (d *DevCycleLocalBucketing) GenerateBucketedConfigForUser(token, user string) (ret BucketedUserConfig, err error) {
	tokenAddr, err := d.newAssemblyScriptString(token)
	if err != nil {
		return
	}
	userAddr, err := d.newAssemblyScriptString(user)
	if err != nil {
		return
	}
	_generateBucketedConfigForUser := d.wasmInstance.GetExport(d.wasmStore, "generateBucketedConfigForUser").Func()
	configPtr, err := _generateBucketedConfigForUser.Call(d.wasmStore, tokenAddr, userAddr)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(readAssemblyScriptString(configPtr.(int32), d.wasmMemory, d.wasmStore)), &ret)
	if err != nil {
		return
	}
	return ret, nil
}

func (d *DevCycleLocalBucketing) StoreConfig(token, config string) error {

	tokenAddr, err := d.newAssemblyScriptString(token)
	if err != nil {
		return err
	}
	configAddr, err := d.newAssemblyScriptString(config)
	if err != nil {
		return err
	}
	_setConfigData := d.wasmInstance.GetExport(d.wasmStore, "setConfigData").Func()
	_, err = _setConfigData.Call(d.wasmStore, tokenAddr, configAddr)
	if err != nil {
		return err
	}
	return nil
}

func (d *DevCycleLocalBucketing) SetPlatformData(platformData string) error {
	configAddr, err := d.newAssemblyScriptString(platformData)
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

// This has a horrible hack because of utf16 - We're double-allocating because utf8->utf16 doesn't zero-padded
// after the first character byte, so we do that manually.
func (d *DevCycleLocalBucketing) newAssemblyScriptString(param string) (int32, error) {
	const objectIdString int32 = 1
	encoded := utf16.Encode([]rune(param))

	__new := d.wasmInstance.GetExport(d.wasmStore, "__new").Func()

	ptr, err := __new.Call(d.wasmStore, int32(len(encoded)*2), objectIdString)
	if err != nil {
		return -1, err
	}
	addr := ptr.(int32)
	var i int32 = 0
	for i = 0; i < int32(len(encoded)); i++ {
		d.wasmMemory.UnsafeData(d.wasmStore)[addr+(i*2)] = byte(encoded[i])
	}
	return ptr.(int32), nil
}

// https://www.assemblyscript.org/runtime.html#memory-layout
// This has a horrible hack of skipping every other index in the resulting array because
// there isn't a great way to parse UTF-16 cleanly that matches the WTF-16 format that WASM uses.
func readAssemblyScriptString(pointer int32, memory *wasmtime.Memory, store *wasmtime.Store) (ret string) {
	stringLength := byteArrayToInt(memory.UnsafeData(store)[pointer-4 : pointer])
	rawData := memory.UnsafeData(store)[pointer : pointer+int32(stringLength)]

	for i := 0; i < len(rawData); i += 2 {
		ret += string(rawData[i])
	}

	return
}

func byteArrayToInt(arr []byte) int64 {
	val := int64(0)
	size := len(arr)
	for i := 0; i < size; i++ {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}
	return val
}
