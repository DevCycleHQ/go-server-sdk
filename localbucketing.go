package devcycle

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"math/rand"
	"os"
	"time"
	"unicode/utf16"

	"sync"
)

var (
	errorMessage = ""
)

type VariableTypeCode int32

type VariableTypeCodes struct {
	Boolean VariableTypeCode
	Number  VariableTypeCode
	String  VariableTypeCode
	JSON    VariableTypeCode
}

type DevCycleLocalBucketing struct {
	wasm          []byte
	eventQueue    *EventQueue
	sdkKey        string
	options       *DVCOptions
	cfg           *HTTPConfiguration
	flushMutex    sync.Mutex
	sdkKeyAddr    uint64
	wazeroRuntime wazero.Runtime
	wazeroMemory  api.Memory
	wazeroModule  api.Module
	wasmMutex     sync.Mutex

	// Cache function pointers
	__newFunc     api.Function
	__unpinFunc   api.Function
	__collectFunc api.Function
	__pinFunc     api.Function

	flushEventQueueFunc               api.Function
	eventQueueSizeFunc                api.Function
	onPayloadSuccessFunc              api.Function
	queueEventFunc                    api.Function
	onPayloadFailureFunc              api.Function
	generateBucketedConfigForUserFunc api.Function
	setPlatformDataFunc               api.Function
	setConfigDataFunc                 api.Function
	initEventQueueFunc                api.Function
	queueAggregateEventFunc           api.Function
	setClientCustomDataFunc           api.Function
	
	variableForUserFunc api.Function
	
	VariableTypeCodes VariableTypeCodes
}

//go:embed bucketing-lib.release.wasm
var wasmBinary []byte

func (d *DevCycleLocalBucketing) Initialize(ctx context.Context, sdkKey string, options *DVCOptions, cfg *HTTPConfiguration) (err error) {
	options.CheckDefaults()

	d.options = options
	d.cfg = cfg
	d.wasm = wasmBinary
	d.eventQueue = &EventQueue{}

	d.wazeroRuntime = wazero.NewRuntime(ctx)
	ascModuleBuilder := d.wazeroRuntime.NewHostModuleBuilder("env")

	ascModuleBuilder.NewFunctionBuilder().WithName("~lib/builtins/seed").
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			r := rand.Float64() * float64(time.Now().UnixMilli())

			// the caller interprets the result as a float64
			stack[0] = uint64(r)
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeF64}).Export("seed")

	ascModuleBuilder.NewFunctionBuilder().WithName("~lib/bindings/dom/Date.now").
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			r := float64(time.Now().UnixMilli())
			// the caller interprets the result as a float64
			stack[0] = uint64(r)
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeF64}).
		Export("Date.now")

	ascModuleBuilder.NewFunctionBuilder().WithName("~lib/builtins/abort").
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			mem := mod.Memory()

			message := uint32(stack[0])
			fileName := uint32(stack[1])
			lineNumber := uint32(stack[2])
			columnNumber := uint32(stack[3])

			if msg, msgErr := readAssemblyScriptString(mem, message); msgErr == nil {
				if fn, fnErr := readAssemblyScriptString(mem, fileName); fnErr == nil {
					errorMessage = fmt.Sprintf("%s at %s:%d:%d\n", msg, fn, lineNumber, columnNumber)
					errorf(errorMessage)
				}
			}
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("message", "fileName", "lineNumber", "columnNumber").
		Export("abort")

	ascModuleBuilder.NewFunctionBuilder().WithName("~lib/bindings/dom/console.log").WithGoModuleFunction(
		api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			mem := mod.Memory()

			message := uint32(stack[0])

			if msg, msgErr := readAssemblyScriptString(mem, message); msgErr == nil {
				printf(msg)
			}
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{}).Export("console.log")

	_, err = ascModuleBuilder.Instantiate(ctx)
	if err != nil {
		return
	}

	d.wazeroModule, err = d.wazeroRuntime.InstantiateWithConfig(context.Background(), d.wasm, wazero.NewModuleConfig().WithStdout(os.Stdout).WithStderr(os.Stderr))
	if err != nil {
		return
	}

	d.initEventQueueFunc = d.wazeroModule.ExportedFunction("initEventQueue")

	d.wazeroMemory = d.wazeroModule.Memory()
	d.flushEventQueueFunc = d.wazeroModule.ExportedFunction("flushEventQueue")
	d.eventQueueSizeFunc = d.wazeroModule.ExportedFunction("eventQueueSize")
	d.onPayloadSuccessFunc = d.wazeroModule.ExportedFunction("onPayloadSuccess")
	d.onPayloadFailureFunc = d.wazeroModule.ExportedFunction("onPayloadFailure")
	d.generateBucketedConfigForUserFunc = d.wazeroModule.ExportedFunction("generateBucketedConfigForUser")
	d.queueEventFunc = d.wazeroModule.ExportedFunction("queueEvent")
	d.queueAggregateEventFunc = d.wazeroModule.ExportedFunction("queueAggregateEvent")
	d.setPlatformDataFunc = d.wazeroModule.ExportedFunction("setPlatformData")
	d.setConfigDataFunc = d.wazeroModule.ExportedFunction("setConfigData")
	d.setClientCustomDataFunc = d.wazeroModule.ExportedFunction("setClientCustomData")

	d.__newFunc = d.wazeroModule.ExportedFunction("__new")
	d.__pinFunc = d.wazeroModule.ExportedFunction("__pin")
	d.__unpinFunc = d.wazeroModule.ExportedFunction("__unpin")
	d.__collectFunc = d.wazeroModule.ExportedFunction("__collect")

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
	err = d.SetPlatformData(ctx, platformJSON)
	if err != nil {
		return
	}

	if err != nil {
		return
	}
	err = d.eventQueue.initialize(ctx, options, d)
	if err != nil {
		return
	}

	return
}

func (d *DevCycleLocalBucketing) setSDKKey(ctx context.Context, sdkKey string) (err error) {
	addr, err := d.newAssemblyScriptString(ctx, []byte(sdkKey))
	if err != nil {
		return
	}

	err = d.assemblyScriptPin(ctx, addr)
	if err != nil {
		return
	}

	d.sdkKey = sdkKey
	d.sdkKeyAddr = uint64(addr)
	return
}

func (d *DevCycleLocalBucketing) initEventQueue(ctx context.Context, options []byte) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	optionsAddr, err := d.newAssemblyScriptString(ctx, options)
	if err != nil {
		return
	}

	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}

	_, err = d.initEventQueueFunc.Call(ctx, d.sdkKeyAddr, uint64(optionsAddr))
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
func (d *DevCycleLocalBucketing) flushEventQueue(ctx context.Context) (payload []FlushPayload, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	addrResult, err := d.flushEventQueueFunc.Call(ctx, d.sdkKeyAddr)
	if err != nil {
		return
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	result, err := readAssemblyScriptString(d.wazeroMemory, uint32(addrResult[0]))
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(result), &payload)
	return
}

func (d *DevCycleLocalBucketing) checkEventQueueSize(ctx context.Context) (length int, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	result, err := d.eventQueueSizeFunc.Call(ctx, d.sdkKeyAddr)
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	if err != nil {
		return
	}
	queueLen := result[0]
	return int(queueLen), nil
}

func (d *DevCycleLocalBucketing) onPayloadSuccess(ctx context.Context, payloadId string) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	payloadIdAddr, err := d.newAssemblyScriptString(ctx, []byte(payloadId))
	if err != nil {
		return
	}

	_, err = d.onPayloadSuccessFunc.Call(ctx, d.sdkKeyAddr, uint64(payloadIdAddr))
	if err != nil || errorMessage != "" {
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
		return
	}
	return
}

func (d *DevCycleLocalBucketing) queueEvent(ctx context.Context, user, event []byte) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	userAddr, err := d.newAssemblyScriptString(ctx, user)
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
	eventAddr, err := d.newAssemblyScriptString(ctx, event)
	if err != nil {
		return
	}

	_, err = d.queueEventFunc.Call(ctx, d.sdkKeyAddr, uint64(userAddr), uint64(eventAddr))
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return
}

func (d *DevCycleLocalBucketing) queueAggregateEvent(ctx context.Context, event []byte, config BucketedUserConfig) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

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
	eventAddr, err := d.newAssemblyScriptString(ctx, event)
	if err != nil {
		return
	}

	_, err = d.queueAggregateEventFunc.Call(ctx, d.sdkKeyAddr, uint64(eventAddr), uint64(variationMapAddr))
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return
}

func (d *DevCycleLocalBucketing) onPayloadFailure(ctx context.Context, payloadId string, retryable bool) (err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	payloadIdAddr, err := d.newAssemblyScriptString(ctx, []byte(payloadId))
	if err != nil {
		return
	}

	if err != nil {
		return
	}

	if retryable {
		_, err = d.onPayloadFailureFunc.Call(ctx, uint64(payloadIdAddr), 1)
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
	} else {
		_, err = d.onPayloadFailureFunc.Call(ctx, d.sdkKeyAddr, uint64(payloadIdAddr), 0)
		if errorMessage != "" {
			err = fmt.Errorf(errorMessage)
		}
	}
	return
}

func (d *DevCycleLocalBucketing) GenerateBucketedConfigForUser(ctx context.Context, user []byte) (ret BucketedUserConfig, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()
	userAddr, err := d.newAssemblyScriptString(ctx, user)
	if err != nil {
		return
	}

	configPtr, err := d.generateBucketedConfigForUserFunc.Call(ctx, d.sdkKeyAddr, uint64(userAddr))
	if err != nil {
		return
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	rawConfig, err := readAssemblyScriptString(d.wazeroMemory, uint32(configPtr[0]))
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(rawConfig), &ret)
	return ret, err
}

func (d *DevCycleLocalBucketing) VariableForUser(ctx context.Context, user []byte, key string, variableType VariableTypeCode) (ret Variable, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()
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

	varPtr, err := d.variableForUserFunc.Call(ctx, d.sdkKeyAddr, uint64(userAddr), uint64(keyAddr), uint64(variableType))
	if err != nil {
		return
	}

	var intPtr = uint32(varPtr[0])

	if intPtr == 0 {
		ret = Variable{}
		return
	}

	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	rawVar, err := d.readAssemblyScriptString(d.wazeroMemory, intPtr)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(rawVar), &ret)
	return ret, err
}

func (d *DevCycleLocalBucketing) StoreConfig(ctx context.Context, config []byte) error {
	defer func() {
		if err := recover(); err != nil {
			errorf("Failed to process config: ", err)
		}
	}()
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	configAddr, err := d.newAssemblyScriptString(ctx, config)
	if err != nil {
		return err
	}

	_, err = d.setConfigDataFunc.Call(ctx, d.sdkKeyAddr, uint64(configAddr))
	if err != nil {
		return err
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return err
}

func (d *DevCycleLocalBucketing) SetPlatformData(ctx context.Context, platformData []byte) error {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	configAddr, err := d.newAssemblyScriptString(ctx, platformData)
	if err != nil {
		return err
	}

	_, err = d.setPlatformDataFunc.Call(ctx, uint64(configAddr))
	if err != nil {
		return err
	}
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return err
}

func (d *DevCycleLocalBucketing) SetClientCustomData(ctx context.Context, customData []byte) error {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	customDataAddr, err := d.newAssemblyScriptString(ctx, customData)
	if err != nil {
		return err
	}

	_, err = d.setClientCustomDataFunc.Call(ctx, d.sdkKeyAddr, uint64(customDataAddr))
	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
	}
	return err
}

// Due to WTF-16, we're double-allocating because utf8 -> utf16 doesn't zero-pad
// after the first character byte, so we do that manually.
func (d *DevCycleLocalBucketing) newAssemblyScriptString(ctx context.Context, param []byte) (uint32, error) {
	const objectIdString uint64 = 2

	// malloc
	ptr, err := d.__newFunc.Call(ctx, uint64(len(param)*2), objectIdString)
	if err != nil {
		return 0, err
	}
	addr := uint32(ptr[0])

	for i, c := range param {
		d.wazeroMemory.WriteByte(addr+uint32(i*2), c)
	}

	if addr == 0 {
		return 0, errorf("Failed to allocate memory for string")
	}
	return addr, nil
}

func (d *DevCycleLocalBucketing) assemblyScriptPin(ctx context.Context, pointer uint32) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptPin - cannot pin")
	}
	_, err = d.__pinFunc.Call(ctx, uint64(pointer))
	if err != nil {
		return err
	}
	return nil
}

func (d *DevCycleLocalBucketing) assemblyScriptCollect(ctx context.Context) (err error) {
	_, err = d.__collectFunc.Call(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (d *DevCycleLocalBucketing) assemblyScriptUnpin(ctx context.Context, pointer uint32) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptUnpin - cannot unpin")
	}

	_, err = d.__unpinFunc.Call(ctx, uint64(pointer))
	if err != nil {
		return err
	}
	return nil
}

// readAssemblyScriptString reads a UTF-16 string created by AssemblyScript.
func (d *DevCycleLocalBucketing) readAssemblyScriptString(mem api.Memory, offset uint32) (string, error) {
	if offset <= 0 {
		return "", errorf("null pointer passed to readAssemblyScriptString - cannot read string")
	}
	// Length is four bytes before pointer.
	byteCount, ok := mem.ReadUint32Le(offset - 4)
	if !ok || byteCount%2 != 0 {
		return "", errorf("invalid string length")
	}
	buf, ok := mem.Read(offset, byteCount)
	if !ok {
		return "", errorf("failed to read string from memory")
	}
	return decodeUTF16(buf), nil
}

func readAssemblyScriptString(mem api.Memory, offset uint32) (string, error) {
	if offset <= 0 {
		return "", errorf("null pointer passed to readAssemblyScriptString - cannot read string")
	}
	// Length is four bytes before pointer.
	byteCount, ok := mem.ReadUint32Le(offset - 4)
	if !ok || byteCount%2 != 0 {
		return "", errorf("invalid string length")
	}
	buf, ok := mem.Read(offset, byteCount)
	if !ok {
		return "", errorf("failed to read string from memory")
	}
	return decodeUTF16(buf), nil
}

func decodeUTF16(b []byte) string {
	u16s := make([]uint16, len(b)/2)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[i/2] = uint16(b[i]) + (uint16(b[i+1]) << 8)
	}

	return string(utf16.Decode(u16s))
}
