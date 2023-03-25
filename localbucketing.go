package devcycle

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/devcyclehq/go-server-sdk/v2/proto"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed bucketing-lib.release.wasm
var wasmBinary []byte

const (
	memoryBucketOffset = 5
	bufferHeaderSize   = 12
)

type VariableTypeCode int32

type VariableTypeCodes struct {
	Boolean VariableTypeCode
	Number  VariableTypeCode
	String  VariableTypeCode
	JSON    VariableTypeCode
}

type DevCycleLocalBucketing struct {
	ctx           context.Context
	wasm          []byte
	sdkKey        string
	options       *DVCOptions
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
	variableForUserFunc               api.Function
	variableForUser_PBFunc            api.Function

	VariableTypeCodes VariableTypeCodes

	// Holds pointers to pre-allocated blocks of memory.
	allocatedMemPool []int32
	// Ptr to reserved block for byte buffer header
	byteBufferHeader int32
	errorMessage     string
}

func (d *DevCycleLocalBucketing) Initialize(sdkKey string, options *DVCOptions) (err error) {
	d.options = options
	d.wasm = wasmBinary
	d.ctx = context.Background()

	// TODO: move wazero runtime back into wasm_main.go
	d.wazeroRuntime = wazero.NewRuntimeWithConfig(d.ctx, wazero.NewRuntimeConfigCompiler())
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

			if msg, msgErr := d.readAssemblyScriptString(mem, message); msgErr == nil {
				if fn, fnErr := d.readAssemblyScriptString(mem, fileName); fnErr == nil {
					d.errorMessage = fmt.Sprintf("WASM Error: %s at %s:%d:%d\n", msg, fn, lineNumber, columnNumber)
					errorf(d.errorMessage)
				}
			}
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("message", "fileName", "lineNumber", "columnNumber").
		Export("abort")

	ascModuleBuilder.NewFunctionBuilder().WithName("~lib/bindings/dom/console.log").WithGoModuleFunction(
		api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			mem := mod.Memory()

			message := uint32(stack[0])

			if msg, msgErr := d.readAssemblyScriptString(mem, message); msgErr == nil {
				printf(msg)
			}
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{}).Export("console.log")

	_, err = ascModuleBuilder.Instantiate(context.Background())
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
	d.variableForUserFunc = d.wazeroModule.ExportedFunction("variableForUserPreallocated")
	d.variableForUser_PBFunc = d.wazeroModule.ExportedFunction("variableForUser_PB_Preallocated")

	d.__newFunc = d.wazeroModule.ExportedFunction("__new")
	d.__pinFunc = d.wazeroModule.ExportedFunction("__pin")
	d.__unpinFunc = d.wazeroModule.ExportedFunction("__unpin")
	d.__collectFunc = d.wazeroModule.ExportedFunction("__collect")

	boolType := int32(d.wazeroModule.ExportedGlobal("VariableType.Boolean").Get())
	numberType := int32(d.wazeroModule.ExportedGlobal("VariableType.Number").Get())
	jsonType := int32(d.wazeroModule.ExportedGlobal("VariableType.JSON").Get())
	stringType := int32(d.wazeroModule.ExportedGlobal("VariableType.String").Get())

	d.VariableTypeCodes = VariableTypeCodes{
		Boolean: VariableTypeCode(boolType),
		String:  VariableTypeCode(stringType),
		Number:  VariableTypeCode(numberType),
		JSON:    VariableTypeCode(jsonType),
	}

	d.allocatedMemPool = make([]int32, d.options.MaxMemoryAllocationBuckets)

	ptr, err := d.allocMemForBuffer(bufferHeaderSize, 9, true)

	if err != nil {
		return err
	}

	// Allocate new memory for the header
	// Format is
	// 4 bytes: pointer address in LE to buffer
	// 4 bytes: pointer address in LE to buffer
	// 4 bytes: length of the buffer in LE
	d.byteBufferHeader = ptr

	// preallocate "buckets" of memory to write data buffers of different lengths to
	// allocate 2^5 bytes to 2^(5+MaxMemoryAllocationBuckets) bytes
	for i := memoryBucketOffset; i < d.options.MaxMemoryAllocationBuckets+memoryBucketOffset; i++ {
		index := i - memoryBucketOffset
		size := 1 << i
		ptr, err := d.allocMemForBuffer(int32(size), 1, true)
		if err != nil {
			return err
		}
		d.allocatedMemPool[index] = ptr
	}

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
	err = d.SetPlatformData(platformJSON)

	return
}

func (d *DevCycleLocalBucketing) setSDKKey(sdkKey string) (err error) {
	addr, err := d.newAssemblyScriptString([]byte(sdkKey))
	if err != nil {
		return
	}

	err = d.assemblyScriptPin(addr)
	if err != nil {
		return
	}

	d.sdkKey = sdkKey
	d.sdkKeyAddr = uint64(addr)
	return
}

func (d *DevCycleLocalBucketing) initEventQueue(options []byte) (err error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	optionsAddr, err := d.newAssemblyScriptString(options)
	if err != nil {
		return
	}

	_, err = d.initEventQueueFunc.Call(d.ctx, d.sdkKeyAddr, uint64(optionsAddr))
	err = d.handleWASMErrors("initEventQueue", err)
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
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	addrResult, err := d.flushEventQueueFunc.Call(d.ctx, d.sdkKeyAddr)
	err = d.handleWASMErrors("flushEventQueue", err)
	if err != nil {
		return
	}
	result, err := d.readAssemblyScriptString(d.wazeroMemory, uint32(addrResult[0]))
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(result), &payload)
	return
}

func (d *DevCycleLocalBucketing) checkEventQueueSize() (length int, err error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	result, err := d.eventQueueSizeFunc.Call(d.ctx, d.sdkKeyAddr)
	err = d.handleWASMErrors("eventQueueSize", err)
	if err != nil {
		return
	}
	queueLen := result[0]
	return int(queueLen), nil
}

func (d *DevCycleLocalBucketing) OnPayloadSuccess(payloadId string) (err error) {
	d.wasmMutex.Lock()
	defer d.wasmMutex.Unlock()

	return d.onPayloadSuccess(payloadId)
}

func (d *DevCycleLocalBucketing) onPayloadSuccess(payloadId string) (err error) {
	d.errorMessage = ""
	payloadIdAddr, err := d.newAssemblyScriptString([]byte(payloadId))
	if err != nil {
		return
	}

	_, err = d.onPayloadSuccessFunc.Call(d.ctx, d.sdkKeyAddr, uint64(payloadIdAddr))
	err = d.handleWASMErrors("onPayloadSuccess", err)
	return
}

func (d *DevCycleLocalBucketing) queueEvent(user, event []byte) (err error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	userAddr, err := d.newAssemblyScriptString([]byte(user))
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
	eventAddr, err := d.newAssemblyScriptString([]byte(event))
	if err != nil {
		return
	}

	_, err = d.queueEventFunc.Call(d.ctx, d.sdkKeyAddr, uint64(userAddr), uint64(eventAddr))
	err = d.handleWASMErrors("queueEvent", err)
	return
}

func (d *DevCycleLocalBucketing) queueAggregateEvent(event []byte, config BucketedUserConfig) (err error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	variationMap, err := json.Marshal(config.VariableVariationMap)
	if err != nil {
		return
	}
	variationMapAddr, err := d.newAssemblyScriptString(variationMap)
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

	_, err = d.queueAggregateEventFunc.Call(d.ctx, d.sdkKeyAddr, uint64(eventAddr), uint64(variationMapAddr))
	err = d.handleWASMErrors("queueAggregateEvent", err)
	return
}

func (d *DevCycleLocalBucketing) OnPayloadFailure(payloadId string, retryable bool) (err error) {
	d.wasmMutex.Lock()
	defer d.wasmMutex.Unlock()

	return d.onPayloadFailure(payloadId, retryable)
}

func (d *DevCycleLocalBucketing) onPayloadFailure(payloadId string, retryable bool) (err error) {
	d.errorMessage = ""

	payloadIdAddr, err := d.newAssemblyScriptString([]byte(payloadId))
	if err != nil {
		return
	}

	if err != nil {
		return
	}

	if retryable {
		_, err = d.onPayloadFailureFunc.Call(d.ctx, d.sdkKeyAddr, uint64(payloadIdAddr), 1)
		err = d.handleWASMErrors("onPayloadFailure", err)
	} else {
		_, err = d.onPayloadFailureFunc.Call(d.ctx, d.sdkKeyAddr, uint64(payloadIdAddr), 0)
		err = d.handleWASMErrors("onPayloadFailure", err)
	}
	return
}

func (d *DevCycleLocalBucketing) GenerateBucketedConfigForUser(user []byte) (ret BucketedUserConfig, err error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()
	userAddr, err := d.newAssemblyScriptString(user)
	if err != nil {
		return
	}

	configPtr, err := d.generateBucketedConfigForUserFunc.Call(d.ctx, d.sdkKeyAddr, uint64(userAddr))
	err = d.handleWASMErrors("generateBucketedConfig", err)
	if err != nil {
		return
	}
	rawConfig, err := d.readAssemblyScriptString(d.wazeroMemory, uint32(configPtr[0]))
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(rawConfig), &ret)
	return ret, err
}

/*
 * This is a helper function to call the variableForUserPB function in the WASM module.
 * It takes a serialized protobuf message as input and returns a serialized protobuf message as output.
 */
func (d *DevCycleLocalBucketing) VariableForUser_PB(serializedParams []byte) (*proto.SDKVariable_PB, error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	paramsAddr, err := d.newAssemblyScriptByteArray(d.wazeroMemory, serializedParams)

	if err != nil {
		return nil, errorf("Error allocating WASM string: %w", err)
	}

	varPtr, err := d.variableForUser_PBFunc.Call(d.ctx, uint64(paramsAddr), uint64(len(serializedParams)))

	err = d.handleWASMErrors("variableForUserPB", err)

	if err != nil {
		return nil, err
	}

	var intPtr = uint32(varPtr[0])

	if intPtr == 0 {
		return nil, nil
	}

	rawVar, err := d.readAssemblyScriptByteArray(d.wazeroMemory, intPtr)
	if err != nil {
		return nil, errorf("Error converting WASM result to bytes: %w", err)
	}

	sdkVariable := proto.SDKVariable_PB{}
	err = sdkVariable.UnmarshalVT(rawVar)

	if err != nil {
		return nil, errorf("Error deserializing WASM result: %w", err)
	}

	return &sdkVariable, nil
}

func (d *DevCycleLocalBucketing) StoreConfig(config []byte) error {
	defer func() {
		if err := recover(); err != nil {
			errorf("Failed to process config: ", err)
		}
	}()
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	configAddr, err := d.newAssemblyScriptString(config)
	if err != nil {
		return err
	}

	_, err = d.setConfigDataFunc.Call(d.ctx, d.sdkKeyAddr, uint64(configAddr))
	err = d.handleWASMErrors("setConfigData", err)

	return err
}

func (d *DevCycleLocalBucketing) SetPlatformData(platformData []byte) error {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	configAddr, err := d.newAssemblyScriptString(platformData)
	if err != nil {
		return err
	}

	_, err = d.setPlatformDataFunc.Call(d.ctx, uint64(configAddr))
	err = d.handleWASMErrors("setPlatformData", err)
	return err
}

func (d *DevCycleLocalBucketing) SetClientCustomData(customData []byte) error {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	customDataAddr, err := d.newAssemblyScriptString(customData)
	if err != nil {
		return err
	}

	_, err = d.setClientCustomDataFunc.Call(d.ctx, d.sdkKeyAddr, uint64(customDataAddr))
	err = d.handleWASMErrors("setClientCustomData", err)
	return err
}

func (d *DevCycleLocalBucketing) HandleFlushResults(result *FlushResult) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	for _, payloadId := range result.SuccessPayloads {
		if err := d.onPayloadSuccess(payloadId); err != nil {
			_ = errorf("failed to mark event payloads as successful", err)
		}
	}
	for _, payloadId := range result.FailurePayloads {
		if err := d.onPayloadFailure(payloadId, false); err != nil {
			_ = errorf("failed to mark event payloads as failed", err)

		}
	}
	for _, payloadId := range result.FailureWithRetryPayloads {
		if err := d.onPayloadFailure(payloadId, true); err != nil {
			_ = errorf("failed to mark event payloads as failed", err)
		}
	}

	return
}

// Due to WTF-16, we're double-allocating because utf8 -> utf16 doesn't zero-pad
// after the first character byte, so we do that manually.
func (d *DevCycleLocalBucketing) newAssemblyScriptString(param []byte) (uint32, error) {
	const objectIdString uint64 = 2

	// malloc
	ptr, err := d.__newFunc.Call(d.ctx, uint64(len(param)*2), objectIdString)
	err = d.handleWASMErrors("__new (newAssemblyScriptString)", err)
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

func (d *DevCycleLocalBucketing) allocMemForBufferPool(size int32) (addr int32, err error) {
	if len(d.allocatedMemPool) == 0 {
		// dont use the pool, fall through to alloc below
	} else {
		// index is the highest power value of 2 for the size we want, offset by the start of the allocation sizes
		cachedIdx := int32(math.Max(memoryBucketOffset, math.Ceil(math.Log2(float64(size))))) - memoryBucketOffset
		// if this index exceeds the max size of the pool, we'll just allocate the memory temporarily
		if cachedIdx >= int32(len(d.allocatedMemPool)) {
			warnf("String size exceeds max memory pool size, allocating new temporary block")
		} else {
			return d.allocatedMemPool[cachedIdx], nil
		}
	}

	// malloc
	ptr, err := d.allocMemForBuffer(size, 1, false)

	return ptr, err
}

func (d *DevCycleLocalBucketing) allocMemForBuffer(size int32, classId int32, shouldPin bool) (addr int32, err error) {
	// malloc
	result, err := d.__newFunc.Call(d.ctx, uint64(size), uint64(classId))
	ptr := uint32(result[0])
	err = d.handleWASMErrors("__new (allocMemForBuffer)", err)
	if err != nil {
		return -1, err
	}
	if shouldPin {
		if err := d.assemblyScriptPin(ptr); err != nil {
			return -1, err
		}
	}
	return int32(ptr), nil
}

func (d *DevCycleLocalBucketing) newAssemblyScriptByteArray(mem api.Memory, param []byte) (int32, error) {
	const align = 0
	length := int32(len(param))

	buffer, err := d.allocMemForBufferPool(length << align)
	// Allocate the full buffer of our data - this is a buffer
	if err != nil {
		return -1, err
	}

	// TODO: The rest of this method can likely be done way more easily with the api.Memory helper methods
	// I just translated the operations to match to avoid mistakes

	// Create a binary buffer to write little endian format
	littleEndianBufferAddress := bytes.NewBuffer([]byte{})

	err = binary.Write(littleEndianBufferAddress, binary.LittleEndian, buffer)
	if err != nil {
		return 0, err
	}

	// Write to the first 8 bytes of the header
	for i, c := range littleEndianBufferAddress.Bytes() {
		mem.WriteByte(uint32(d.byteBufferHeader+int32(i)), c)
		mem.WriteByte(uint32(d.byteBufferHeader+int32(i)+4), c)
	}

	// Create another binary buffer to write the length of the buffer
	lengthBuffer := bytes.NewBuffer([]byte{})
	err = binary.Write(lengthBuffer, binary.LittleEndian, length<<align)
	if err != nil {
		return 0, err
	}
	// Write the length to the last 4 bytes of the header
	for i, c := range lengthBuffer.Bytes() {
		mem.WriteByte(uint32(d.byteBufferHeader+int32(i)+8), c)
	}

	// Write the buffer itself into WASM.
	for i, c := range param {
		mem.WriteByte(uint32(buffer+int32(i)), c)
	}

	// Return the header address - as that's what's consumed on the WASM side.
	return d.byteBufferHeader, nil
}

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

func (d *DevCycleLocalBucketing) readAssemblyScriptByteArray(mem api.Memory, offset uint32) ([]byte, error) {
	if offset <= 0 {
		return nil, errorf("null pointer passed to readAssemblyScriptByteArray - cannot read string")
	}

	// Length is 8 bytes after offset.
	byteCount, ok := mem.ReadUint32Le(offset + 8)
	if !ok {
		return nil, errorf("readAssemblyScriptByteArray: failed to read length of byte array: %v", byteCount)
	}

	// Data pointer is first 4 bytes after offset.
	dataPointer, ok := mem.ReadUint32Le(offset)
	if !ok {
		return nil, errorf("readAssemblyScriptByteArray: failed to read byte array data pointer")
	}

	buf, ok := mem.Read(dataPointer, byteCount)
	if !ok {
		return nil, errorf("readAssemblyScriptByteArray: failed to read byte array from memory")
	}

	return buf, nil
}

func (d *DevCycleLocalBucketing) assemblyScriptPin(pointer uint32) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptPin - cannot pin")
	}
	_, err = d.__pinFunc.Call(d.ctx, uint64(pointer))
	err = d.handleWASMErrors("__pin", err)
	return nil
}

func (d *DevCycleLocalBucketing) assemblyScriptCollect() (err error) {
	_, err = d.__collectFunc.Call(d.ctx)
	err = d.handleWASMErrors("__collect", err)
	return nil
}

func (d *DevCycleLocalBucketing) assemblyScriptUnpin(pointer uint32) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptUnpin - cannot unpin")
	}

	_, err = d.__unpinFunc.Call(d.ctx, uint64(pointer))
	err = d.handleWASMErrors("__unpin", err)
	return
}

func (d *DevCycleLocalBucketing) handleWASMErrors(prefix string, err error) error {
	if d.errorMessage != "" {
		if err != nil {
			return errorf(
				"Error Message calling %s: err: [%s] errorMessage:[%s]",
				prefix,
				strings.ReplaceAll(err.Error(), "\n", ""),
				d.errorMessage,
			)
		}
		return errorf(d.errorMessage)
	}

	if err != nil {
		return errorf("Error calling %s: %w", prefix, err)
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

func decodeUTF16(b []byte) string {
	u16s := make([]uint16, len(b)/2)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[i/2] = uint16(b[i]) + (uint16(b[i+1]) << 8)
	}

	return string(utf16.Decode(u16s))
}
