package devcycle

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"sync"

	"github.com/tetratelabs/wazero"
	wasm "github.com/tetratelabs/wazero"
	wasmapi "github.com/tetratelabs/wazero/api"
)

type DevCycleLocalBucketingV2 struct {
	eventQueue *EventQueue
	options    *DVCOptions
	cfg        *HTTPConfiguration
	wasmMutex  sync.Mutex
	flushMutex sync.Mutex
	sdkKeyAddr uint64
	sdkKey     string

	wasm        []byte
	wasmRuntime wasm.Runtime
	wasmModule  wasmapi.Module

	// Cache function pointers
	__newFunc     wasmapi.Function
	__unpinFunc   wasmapi.Function
	__collectFunc wasmapi.Function
	__pinFunc     wasmapi.Function

	flushEventQueueFunc               wasmapi.Function
	eventQueueSizeFunc                wasmapi.Function
	onPayloadSuccessFunc              wasmapi.Function
	queueEventFunc                    wasmapi.Function
	onPayloadFailureFunc              wasmapi.Function
	generateBucketedConfigForUserFunc wasmapi.Function
	setPlatformDataFunc               wasmapi.Function
	setConfigDataFunc                 wasmapi.Function
	initEventQueueFunc                wasmapi.Function
	queueAggregateEventFunc           wasmapi.Function
	setClientCustomDataFunc           wasmapi.Function
	variableForUserFunc               wasmapi.Function

	VariableTypeCodes VariableTypeCodes
}

func (d *DevCycleLocalBucketingV2) Initialize(sdkKey string, options *DVCOptions, cfg *HTTPConfiguration) (err error) {
	options.CheckDefaults()

	// Choose the context to use for function calls.
	ctx := context.Background() // TODO: Pass context as argument

	d.wasmRuntime = wazero.NewRuntime(ctx)
	// TODO: d.wasmRuntime.Close()

	d.options = options
	d.cfg = cfg
	d.wasm = wasmBinary
	d.eventQueue = &EventQueue{}

	////// vvvvvvv------- TODO: rewrite

	// d.wasiConfig = wasmtime.NewWasiConfig()
	// d.wasiConfig.InheritEnv()
	// d.wasiConfig.InheritStderr()
	// d.wasiConfig.InheritStdout()
	// d.wasmStore = wasmtime.NewStore(wasmtime.NewEngine())
	// d.wasmStore.SetWasi(d.wasiConfig)
	// d.wasmLinker = wasmtime.NewLinker(d.wasmStore.Engine)
	// err = d.wasmLinker.DefineWasi()

	// if err != nil {
	// return
	// }

	////// ^^^^^^------- to rewrite

	dateNowFunc := func() float64 { return float64(time.Now().UnixMilli()) }

	abortFunc := func(messagePtr, filenamePointer, lineNum, colNum int32) {
		var errorMessage []byte
		errorMessage, err = d.mallocAssemblyScriptBytes(ctx, uint64(messagePtr))
		if err != nil {
			_ = errorf("WASM Error: %s", err)
			return
		}
		_ = errorf("WASM Error: %s", string(errorMessage))
		err = nil
	}

	consoleLogFunc := func(messagePtr int32) {
		var message []byte
		message, err = d.mallocAssemblyScriptBytes(ctx, uint64(messagePtr))
		printf(string(message))
	}

	seedFunc := func() float64 {
		return rand.Float64() * float64(time.Now().UnixMilli())
	}

	_, err = d.wasmRuntime.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(dateNowFunc).Export("Date.now").
		NewFunctionBuilder().WithFunc(abortFunc).Export("abort").
		NewFunctionBuilder().WithFunc(consoleLogFunc).Export("console.log").
		NewFunctionBuilder().WithFunc(seedFunc).Export("seed").
		Instantiate(ctx)
	if err != nil {
		return err
	}

	d.wasmModule, err = d.wasmRuntime.Instantiate(ctx, wasmBinary)
	if err != nil {
		return err
	}

	d.initEventQueueFunc = d.wasmModule.ExportedFunction("initEventQueue")
	d.flushEventQueueFunc = d.wasmModule.ExportedFunction("flushEventQueue")
	d.eventQueueSizeFunc = d.wasmModule.ExportedFunction("eventQueueSize")
	d.onPayloadSuccessFunc = d.wasmModule.ExportedFunction("onPayloadSuccess")
	d.onPayloadFailureFunc = d.wasmModule.ExportedFunction("onPayloadFailure")
	d.generateBucketedConfigForUserFunc = d.wasmModule.ExportedFunction("generateBucketedConfigForUser")
	d.queueEventFunc = d.wasmModule.ExportedFunction("queueEvent")
	d.queueAggregateEventFunc = d.wasmModule.ExportedFunction("queueAggregateEvent")
	d.setPlatformDataFunc = d.wasmModule.ExportedFunction("setPlatformData")
	d.setClientCustomDataFunc = d.wasmModule.ExportedFunction("setClientCustomData")
	d.setConfigDataFunc = d.wasmModule.ExportedFunction("setConfigData")
	d.variableForUserFunc = d.wasmModule.ExportedFunction("variableForUser")

	// bind exported internal functions
	d.__newFunc = d.wasmModule.ExportedFunction("__new")
	d.__pinFunc = d.wasmModule.ExportedFunction("__pin")
	d.__unpinFunc = d.wasmModule.ExportedFunction("__unpin")
	d.__collectFunc = d.wasmModule.ExportedFunction("__collect")

	boolType := d.wasmModule.ExportedGlobal("VariableType.Boolean").Get()
	stringType := d.wasmModule.ExportedGlobal("VariableType.String").Get()
	numberType := d.wasmModule.ExportedGlobal("VariableType.Number").Get()
	jsonType := d.wasmModule.ExportedGlobal("VariableType.JSON").Get()

	d.VariableTypeCodes = VariableTypeCodes{
		Boolean: VariableTypeCode(boolType),
		String:  VariableTypeCode(stringType),
		Number:  VariableTypeCode(numberType),
		JSON:    VariableTypeCode(jsonType),
	}

	err = d.setSDKKey(ctx, sdkKey)
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

// Getters so I can use both implementations at the same time
func (d *DevCycleLocalBucketingV2) getSDKKey() string {
	return d.sdkKey
}

func (d *DevCycleLocalBucketingV2) getCfg() *HTTPConfiguration {
	return d.cfg
}

func (d *DevCycleLocalBucketingV2) setSDKKey(ctx context.Context, sdkKey string) (err error) {
	addr, err := d.newAssemblyScriptString(ctx, []byte(sdkKey))
	if err != nil {
		return
	}

	if err = d.assemblyScriptPin(ctx, addr); err != nil {
		return err
	}

	d.sdkKeyAddr = uint64(addr)
	d.sdkKey = sdkKey
	return
}

func (d *DevCycleLocalBucketingV2) initEventQueue(options []byte) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	optionsAddr, err := d.newAssemblyScriptString(ctx, options)
	if err != nil {
		return
	}

	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}

	_, err = d.initEventQueueFunc.Call(ctx, d.sdkKeyAddr, optionsAddr)
	if err != nil || errorMessage != "" {
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
		return
	}
	return
}

func (d *DevCycleLocalBucketingV2) startFlushEvents() {
	d.flushMutex.Lock()
}

func (d *DevCycleLocalBucketingV2) finishFlushEvents() {
	d.flushMutex.Unlock()
}

func (d *DevCycleLocalBucketingV2) flushEventQueue() (payload []FlushPayload, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	addrResults, err := d.flushEventQueueFunc.Call(ctx, d.sdkKeyAddr)
	if err != nil {
		return
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	addr := addrResults[0]

	result, err := d.mallocAssemblyScriptBytes(ctx, addr)
	if err != nil {
		return
	}
	err = json.Unmarshal(result, &payload)
	return
}

func (d *DevCycleLocalBucketingV2) checkEventQueueSize() (length int, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	results, err := d.eventQueueSizeFunc.Call(ctx, d.sdkKeyAddr)
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	if err != nil {
		return
	}
	result := results[0]
	return int(result), nil
}

func (d *DevCycleLocalBucketingV2) onPayloadSuccess(payloadId string) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	payloadIdAddr, err := d.newAssemblyScriptString(ctx, []byte(payloadId))
	if err != nil {
		return
	}

	_, err = d.onPayloadSuccessFunc.Call(ctx, d.sdkKeyAddr, payloadIdAddr)
	if err != nil || errorMessage != "" {
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
		return
	}
	return
}

func (d *DevCycleLocalBucketingV2) queueEvent(user, event string) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()

	userAddr, err := d.newAssemblyScriptString(ctx, []byte(user))
	if err != nil {
		return
	}
	err = d.assemblyScriptPin(ctx, userAddr)
	if err != nil {
		return err
	}
	defer func() {
		err := d.assemblyScriptUnpin(ctx, userAddr)
		if err != nil {
			errorf(err.Error())
		}
	}()
	eventAddr, err := d.newAssemblyScriptString(ctx, []byte(event))
	if err != nil {
		return
	}

	_, err = d.queueEventFunc.Call(ctx, d.sdkKeyAddr, userAddr, eventAddr)
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return
}

func (d *DevCycleLocalBucketingV2) queueAggregateEvent(event string, config BucketedUserConfig) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	variationMap, err := json.Marshal(config.VariableVariationMap)
	if err != nil {
		return
	}
	variationMapAddr, err := d.newAssemblyScriptString(ctx, variationMap)
	if err != nil {
		return
	}
	err = d.assemblyScriptPin(ctx, variationMapAddr)
	if err != nil {
		return err
	}
	defer func() {
		err := d.assemblyScriptUnpin(ctx, variationMapAddr)
		if err != nil {
			errorf(err.Error())
		}
	}()
	eventAddr, err := d.newAssemblyScriptString(ctx, []byte(event))
	if err != nil {
		return
	}

	_, err = d.queueAggregateEventFunc.Call(ctx, d.sdkKeyAddr, eventAddr, variationMapAddr)
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return
}

func (d *DevCycleLocalBucketingV2) onPayloadFailure(payloadId string, retryable bool) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	payloadIdAddr, err := d.newAssemblyScriptString(ctx, []byte(payloadId))
	if err != nil {
		return
	}

	if err != nil {
		return
	}

	if retryable {
		_, err = d.onPayloadFailureFunc.Call(ctx, d.sdkKeyAddr, payloadIdAddr, 1)
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
	} else {
		_, err = d.onPayloadFailureFunc.Call(ctx, d.sdkKeyAddr, payloadIdAddr, 0)
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
	}
	return
}

func (d *DevCycleLocalBucketingV2) GenerateBucketedConfigForUser(user string) (ret BucketedUserConfig, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	userAddr, err := d.newAssemblyScriptString(ctx, []byte(user))
	if err != nil {
		return
	}

	configPtrs, err := d.generateBucketedConfigForUserFunc.Call(ctx, d.sdkKeyAddr, userAddr)
	if err != nil {
		return
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}

	configPtr := configPtrs[0]
	rawConfig, err := d.mallocAssemblyScriptBytes(ctx, configPtr)
	if err != nil {
		return
	}
	err = json.Unmarshal(rawConfig, &ret)
	return ret, err
}

func (d *DevCycleLocalBucketingV2) VariableForUser(user []byte, key string, variableType VariableTypeCode) (ret Variable, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	keyAddr, err := d.newAssemblyScriptString(ctx, []byte(key))
	if err != nil {
		return
	}

	err = d.assemblyScriptPin(ctx, keyAddr)
	if err != nil {
		return
	}

	defer func() {
		err := d.assemblyScriptUnpin(ctx, keyAddr)

		if err != nil {
			errorf(err.Error())
		}
	}()

	userAddr, err := d.newAssemblyScriptString(ctx, user)
	if err != nil {
		return
	}

	varPtr, err := d.variableForUserFunc.Call(ctx, d.sdkKeyAddr, userAddr, keyAddr, uint64(variableType))
	if err != nil {
		return
	}

	var intPtr = varPtr[0]

	if intPtr == 0 {
		ret = Variable{}
		return
	}

	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}

	rawVar, err := d.mallocAssemblyScriptBytes(ctx, intPtr)
	if err != nil {
		return
	}
	err = json.Unmarshal(rawVar, &ret)
	if err != nil {
		return ret, err
	}

	return ret, err
}

func (d *DevCycleLocalBucketingV2) StoreConfig(config string) error {
	defer func() {
		if err := recover(); err != nil {
			errorf("Failed to process config: ", err)
		}
	}()
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	configAddr, err := d.newAssemblyScriptString(ctx, []byte(config))
	if err != nil {
		return err
	}

	_, err = d.setConfigDataFunc.Call(ctx, d.sdkKeyAddr, configAddr)
	if err != nil {
		return err
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return err
}

func (d *DevCycleLocalBucketingV2) SetPlatformData(platformData string) error {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	configAddr, err := d.newAssemblyScriptString(ctx, []byte(platformData))
	if err != nil {
		return err
	}

	_, err = d.setPlatformDataFunc.Call(ctx, configAddr)
	if err != nil {
		return err
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return err
}

func (d *DevCycleLocalBucketingV2) SetClientCustomData(customData string) error {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	ctx := context.Background()
	customDataAddr, err := d.newAssemblyScriptString(ctx, []byte(customData))
	if err != nil {
		return err
	}

	_, err = d.setClientCustomDataFunc.Call(ctx, d.sdkKeyAddr, customDataAddr)
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return err
}

// Due to WTF-16, we're double-allocating because utf8 -> utf16 doesn't zero-pad
// after the first character byte, so we do that manually.
func (d *DevCycleLocalBucketingV2) newAssemblyScriptString(ctx context.Context, param []byte) (uint64, error) {
	const objectIdString uint64 = 2
	writeSize := uint64(len(param) * 2)

	// malloc | TODO: do we have to free this memory?
	ptr, err := d.__newFunc.Call(ctx, writeSize, objectIdString)
	if err != nil {
		return 0, err
	}

	addr := uint32(ptr[0])
	if addr == 0 {
		return 0, errorf("Failed to allocate memory for string")
	}

	// The pointer is a linear memory offset, which is where we write the param
	mem := d.wasmModule.Memory()
	for i, c := range param { // TODO: improve performance here
		if !mem.Write(addr+uint32(i*2), []byte{c}) {
			return 0, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d", addr, writeSize, d.wasmModule.Memory().Size())
		}
	}

	return uint64(addr), nil
}

// https://www.assemblyscript.org/runtime.html#memory-layout
// This skips every other index in the resulting array because
// there isn't a great way to parse UTF-16 cleanly that matches the WTF-16 format that ASC uses.
func (d *DevCycleLocalBucketingV2) mallocAssemblyScriptBytes(ctx context.Context, pointer uint64) ([]byte, error) {
	if pointer == 0 {
		return nil, errorf("null pointer passed to mallocAssemblyScriptString - cannot write string")
	}

	mem := d.wasmModule.Memory()
	stringLengthBytes, ok := mem.Read(uint32(pointer-4), 4)
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d", pointer, 4, mem.Size())
	}

	stringLength := byteArrayToInt(stringLengthBytes)
	bytes, ok := mem.Read(uint32(pointer), uint32(stringLength))
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d", pointer, stringLength, mem.Size())
	}

	ret := make([]byte, len(bytes)/2) // TODO: Improve performance
	for i := 0; i < len(bytes); i += 2 {
		ret[i/2] += bytes[i]
	}

	return ret, nil
}

func (d *DevCycleLocalBucketingV2) assemblyScriptPin(ctx context.Context, pointer uint64) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptPin - cannot pin")
	}
	_, err = d.__pinFunc.Call(ctx, pointer)
	if err != nil {
		return err
	}
	return nil
}

func (d *DevCycleLocalBucketingV2) assemblyScriptCollect(ctx context.Context) (err error) {
	_, err = d.__collectFunc.Call(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (d *DevCycleLocalBucketingV2) assemblyScriptUnpin(ctx context.Context, pointer uint64) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptUnpin - cannot unpin")
	}

	_, err = d.__unpinFunc.Call(ctx, pointer)
	if err != nil {
		return err
	}
	return nil
}
