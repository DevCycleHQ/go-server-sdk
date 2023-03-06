package devcycle

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"math/rand"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/bytecodealliance/wasmtime-go/v6"
)

var (
	errorMessage = ""
)

type DevCycleLocalBucketing struct {
	wasm         []byte
	wasmStore    *wasmtime.Store
	wasmModule   *wasmtime.Module
	wasmInstance *wasmtime.Instance
	wasmLinker   *wasmtime.Linker
	wasiConfig   *wasmtime.WasiConfig
	wasmMemory   *wasmtime.Memory
	eventQueue   *EventQueue
	sdkKey       string
	options      *DVCOptions
	cfg          *HTTPConfiguration
	wasmMutex    sync.Mutex
	flushMutex   sync.Mutex
	sdkKeyAddr   int32

	// Cache function pointers
	__newFunc     *wasmtime.Func
	__unpinFunc   *wasmtime.Func
	__collectFunc *wasmtime.Func
	__pinFunc     *wasmtime.Func

	flushEventQueueFunc               *wasmtime.Func
	eventQueueSizeFunc                *wasmtime.Func
	onPayloadSuccessFunc              *wasmtime.Func
	queueEventFunc                    *wasmtime.Func
	onPayloadFailureFunc              *wasmtime.Func
	generateBucketedConfigForUserFunc *wasmtime.Func
	setPlatformDataFunc               *wasmtime.Func
	setConfigDataFunc                 *wasmtime.Func
	initEventQueueFunc                *wasmtime.Func
	queueAggregateEventFunc           *wasmtime.Func
	setClientCustomDataFunc           *wasmtime.Func
}

//go:embed bucketing-lib.release.wasm
var wasmBinary []byte

func (d *DevCycleLocalBucketing) Initialize(sdkKey string, options *DVCOptions, cfg *HTTPConfiguration) (err error) {
	options.CheckDefaults()

	d.options = options
	d.cfg = cfg
	d.wasm = wasmBinary
	d.eventQueue = &EventQueue{}
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
		var errorMessage []byte
		errorMessage, err = d.mallocAssemblyScriptString(messagePtr)
		if err != nil {
			_ = errorf("WASM Error: %s", err)
			return
		}
		_ = errorf("WASM Error: %s", errorMessage)
		err = nil
	})
	if err != nil {
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "console.log", func(messagePtr int32) {
		var errorMessage []byte
		errorMessage, err = d.mallocAssemblyScriptString(messagePtr)
		printf(string(errorMessage))
	})
	if err != nil {
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "seed", func() float64 {
		return rand.Float64() * float64(time.Now().UnixMilli())
	})
	if err != nil {
		return
	}

	d.wasmInstance, err = d.wasmLinker.Instantiate(d.wasmStore, d.wasmModule)
	if err != nil {
		return
	}
	d.wasmMemory = d.wasmInstance.GetExport(d.wasmStore, "memory").Memory()

	// bind exported functions
	d.initEventQueueFunc = d.wasmInstance.GetExport(d.wasmStore, "initEventQueue").Func()
	d.flushEventQueueFunc = d.wasmInstance.GetExport(d.wasmStore, "flushEventQueue").Func()
	d.eventQueueSizeFunc = d.wasmInstance.GetExport(d.wasmStore, "eventQueueSize").Func()
	d.onPayloadSuccessFunc = d.wasmInstance.GetExport(d.wasmStore, "onPayloadSuccess").Func()
	d.onPayloadFailureFunc = d.wasmInstance.GetExport(d.wasmStore, "onPayloadFailure").Func()
	d.generateBucketedConfigForUserFunc = d.wasmInstance.GetExport(d.wasmStore, "generateBucketedConfigForUser").Func()
	d.queueEventFunc = d.wasmInstance.GetExport(d.wasmStore, "queueEvent").Func()
	d.queueAggregateEventFunc = d.wasmInstance.GetExport(d.wasmStore, "queueAggregateEvent").Func()
	d.setPlatformDataFunc = d.wasmInstance.GetExport(d.wasmStore, "setPlatformData").Func()
	d.setClientCustomDataFunc = d.wasmInstance.GetExport(d.wasmStore, "setClientCustomData").Func()
	d.setConfigDataFunc = d.wasmInstance.GetExport(d.wasmStore, "setConfigData").Func()

	// bind exported internal functions
	d.__newFunc = d.wasmInstance.GetExport(d.wasmStore, "__new").Func()
	d.__pinFunc = d.wasmInstance.GetExport(d.wasmStore, "__pin").Func()
	d.__unpinFunc = d.wasmInstance.GetExport(d.wasmStore, "__unpin").Func()
	d.__collectFunc = d.wasmInstance.GetExport(d.wasmStore, "__collect").Func()

	err = d.setSDKKey(sdkKey)
	if err != nil {
		return
	}

	platformData := PlatformData{}
	platformData = *platformData.Default()
	platformJSON, err := json.Marshal(platformData)
	if err != nil {
		return
	}
	err = d.SetPlatformData(string(platformJSON))
	if err != nil {
		return
	}

	if err != nil {
		return
	}
	err = d.eventQueue.initialize(options, d)
	if err != nil {
		return
	}

	return
}

func (d *DevCycleLocalBucketing) setSDKKey(sdkKey string) (err error) {
	addr, err := d.newAssemblyScriptString(sdkKey)
	if err != nil {
		return
	}

	err = d.assemblyScriptPin(addr)
	if err != nil {
		return
	}

	d.sdkKey = sdkKey
	d.sdkKeyAddr = addr
	return
}

func (d *DevCycleLocalBucketing) initEventQueue(options string) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	optionsAddr, err := d.newAssemblyScriptString(options)
	if err != nil {
		return
	}

	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}

	_, err = d.initEventQueueFunc.Call(d.wasmStore, d.sdkKeyAddr, optionsAddr)
	if err != nil || errorMessage != "" {
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
		return
	}
	return
}

func (d *DevCycleLocalBucketing) startFlushEvents() {
	d.flushMutex.Lock()
}

func (d *DevCycleLocalBucketing) finishFlushEvents() {
	d.flushMutex.Unlock()
}
func (d *DevCycleLocalBucketing) flushEventQueue() (payload []FlushPayload, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	addrResult, err := d.flushEventQueueFunc.Call(d.wasmStore, d.sdkKeyAddr)
	if err != nil {
		return
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	result, err := d.mallocAssemblyScriptString(addrResult.(int32))
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(result), &payload)
	return
}

func (d *DevCycleLocalBucketing) checkEventQueueSize() (length int, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	result, err := d.eventQueueSizeFunc.Call(d.wasmStore, d.sdkKeyAddr)
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	if err != nil {
		return
	}
	queueLen := result.(int32)
	return int(queueLen), nil
}

func (d *DevCycleLocalBucketing) onPayloadSuccess(payloadId string) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	payloadIdAddr, err := d.newAssemblyScriptString(payloadId)
	if err != nil {
		return
	}

	_, err = d.onPayloadSuccessFunc.Call(d.wasmStore, d.sdkKeyAddr, payloadIdAddr)
	if err != nil || errorMessage != "" {
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
		return
	}
	return
}

func (d *DevCycleLocalBucketing) queueEvent(user, event string) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	userAddr, err := d.newAssemblyScriptString(user)
	if err != nil {
		return
	}
	err = d.assemblyScriptPin(userAddr)
	if err != nil {
		return err
	}
	defer func() {
		err := d.assemblyScriptUnpin(userAddr)
		if err != nil {
			errorf(err.Error())
		}
	}()
	eventAddr, err := d.newAssemblyScriptString(event)
	if err != nil {
		return
	}

	_, err = d.queueEventFunc.Call(d.wasmStore, d.sdkKeyAddr, userAddr, eventAddr)
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return
}

func (d *DevCycleLocalBucketing) queueAggregateEvent(event string, config BucketedUserConfig) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	variationMap, err := json.Marshal(config.VariableVariationMap)
	if err != nil {
		return
	}
	variationMapAddr, err := d.newAssemblyScriptString(string(variationMap))
	if err != nil {
		return
	}
	err = d.assemblyScriptPin(variationMapAddr)
	if err != nil {
		return err
	}
	defer func() {
		err := d.assemblyScriptUnpin(variationMapAddr)
		if err != nil {
			errorf(err.Error())
		}
	}()
	eventAddr, err := d.newAssemblyScriptString(event)
	if err != nil {
		return
	}

	_, err = d.queueAggregateEventFunc.Call(d.wasmStore, d.sdkKeyAddr, eventAddr, variationMapAddr)
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return
}

func (d *DevCycleLocalBucketing) onPayloadFailure(payloadId string, retryable bool) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	payloadIdAddr, err := d.newAssemblyScriptString(payloadId)
	if err != nil {
		return
	}

	if err != nil {
		return
	}

	if retryable {
		_, err = d.onPayloadFailureFunc.Call(d.wasmStore, d.sdkKeyAddr, payloadIdAddr, 1)
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
	} else {
		_, err = d.onPayloadFailureFunc.Call(d.wasmStore, d.sdkKeyAddr, payloadIdAddr, 0)
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
	}
	return
}

func (d *DevCycleLocalBucketing) GenerateBucketedConfigForUser(user string) (ret BucketedUserConfig, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()
	userAddr, err := d.newAssemblyScriptString(user)
	if err != nil {
		return
	}

	configPtr, err := d.generateBucketedConfigForUserFunc.Call(d.wasmStore, d.sdkKeyAddr, userAddr)
	if err != nil {
		return
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	rawConfig, err := d.mallocAssemblyScriptString(configPtr.(int32))
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(rawConfig), &ret)
	return ret, err
}

func (d *DevCycleLocalBucketing) StoreConfig(config string) error {
	defer func() {
		if err := recover(); err != nil {
			errorf("Failed to process config: ", err)
		}
	}()
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	configAddr, err := d.newAssemblyScriptString(config)
	if err != nil {
		return err
	}

	_, err = d.setConfigDataFunc.Call(d.wasmStore, d.sdkKeyAddr, configAddr)
	if err != nil {
		return err
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return err
}

func (d *DevCycleLocalBucketing) SetPlatformData(platformData string) error {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	configAddr, err := d.newAssemblyScriptString(platformData)
	if err != nil {
		return err
	}

	_, err = d.setPlatformDataFunc.Call(d.wasmStore, configAddr)
	if err != nil {
		return err
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return err
}

func (d *DevCycleLocalBucketing) SetClientCustomData(customData string) error {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	customDataAddr, err := d.newAssemblyScriptString(customData)
	if err != nil {
		return err
	}

	_, err = d.setClientCustomDataFunc.Call(d.wasmStore, d.sdkKeyAddr, customDataAddr)
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return err
}

// Due to WTF-16, we're double-allocating because utf8 -> utf16 doesn't zero-pad
// after the first character byte, so we do that manually.
func (d *DevCycleLocalBucketing) newAssemblyScriptString(param string) (int32, error) {
	const objectIdString int32 = 1
	encoded := utf16.Encode([]rune(param))

	// malloc
	ptr, err := d.__newFunc.Call(d.wasmStore, int32(len(encoded)*2), objectIdString)
	if err != nil {
		return -1, err
	}
	addr := ptr.(int32)
	var i int32 = 0
	data := d.wasmMemory.UnsafeData(d.wasmStore)
	for i = 0; i < int32(len(encoded)); i++ {
		data[addr+(i*2)] = byte(encoded[i])
	}
	dataAddress := ptr.(int32)
	if dataAddress == 0 {
		return -1, errorf("Failed to allocate memory for string")
	}
	return ptr.(int32), nil
}

// https://www.assemblyscript.org/runtime.html#memory-layout
// This skips every other index in the resulting array because
// there isn't a great way to parse UTF-16 cleanly that matches the WTF-16 format that ASC uses.
func (d *DevCycleLocalBucketing) mallocAssemblyScriptString(pointer int32) ([]byte, error) {
	if pointer == 0 {
		return nil, errorf("null pointer passed to mallocAssemblyScriptString - cannot write string")
	}

	data := d.wasmMemory.UnsafeData(d.wasmStore)
	stringLength := byteArrayToInt(data[pointer-4 : pointer])
	rawData := data[pointer : pointer+int32(stringLength)]

	ret := make([]byte, len(rawData)/2)

	for i := 0; i < len(rawData); i += 2 {
		ret[i/2] += rawData[i]
	}

	return ret, nil
}

func (d *DevCycleLocalBucketing) assemblyScriptPin(pointer int32) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptPin - cannot pin")
	}
	_, err = d.__pinFunc.Call(d.wasmStore, pointer)
	if err != nil {
		return err
	}
	return nil
}

func (d *DevCycleLocalBucketing) assemblyScriptCollect() (err error) {
	_, err = d.__collectFunc.Call(d.wasmStore)
	if err != nil {
		return err
	}
	return nil
}

func (d *DevCycleLocalBucketing) assemblyScriptUnpin(pointer int32) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptUnpin - cannot unpin")
	}

	_, err = d.__unpinFunc.Call(d.wasmStore, pointer)
	if err != nil {
		return err
	}
	return nil
}

func byteArrayToInt(arr []byte) int64 {
	val := int64(0)
	size := len(arr)
	for i := 0; i < size; i++ {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}
	return val
}
