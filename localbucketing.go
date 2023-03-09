package devcycle

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
	"unsafe"

	"github.com/bytecodealliance/wasmtime-go/v6"
)

var (
	errorMessage = ""
)

const (
	memoryBucketOffset = 5
)

type VariableTypeCode int32

type VariableTypeCodes struct {
	Boolean VariableTypeCode
	Number  VariableTypeCode
	String  VariableTypeCode
	JSON    VariableTypeCode
}

type DevCycleLocalBucketing struct {
	wasm         []byte
	wasmStore    *wasmtime.Store
	wasmInstance *wasmtime.Instance
	wasmMemory   *wasmtime.Memory
	wasiConfig   *wasmtime.WasiConfig
	wasmMain     *WASMMain
	sdkKey       string
	options      *DVCOptions
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
	variableForUserFunc               *wasmtime.Func

	VariableTypeCodes VariableTypeCodes

	// Holds pointers to pre-allocated blocks of memory. The first dimension is an index based on the size of the data
	// The second dimension is a list of pointers to blocks of memory of that size
	allocatedMemPool [][]int32
}

func (d *DevCycleLocalBucketing) Initialize(wasmMain *WASMMain, sdkKey string, options *DVCOptions) (err error) {
	options.CheckDefaults()

	d.options = options
	d.wasmMain = wasmMain

	d.wasiConfig = wasmtime.NewWasiConfig()
	d.wasiConfig.InheritEnv()
	d.wasiConfig.InheritStderr()
	d.wasiConfig.InheritStdout()

	d.wasmStore = wasmtime.NewStore(d.wasmMain.wasmEngine)
	d.wasmStore.SetWasi(d.wasiConfig)

	if err != nil {
		return
	}

	err = d.wasmMain.wasmLinker.DefineFunc(d.wasmStore, "env", "Date.now", func() float64 { return float64(time.Now().UnixMilli()) })
	if err != nil {
		return
	}

	err = d.wasmMain.wasmLinker.DefineFunc(d.wasmStore, "env", "abort", func(messagePtr, filenamePointer, lineNum, colNum int32) {
		var errorMessage []byte
		errorMessage, err = d.mallocAssemblyScriptBytes(messagePtr)
		if err != nil {
			_ = errorf("WASM Error: %s", err)
			return
		}
		_ = errorf("WASM Error: %s", string(errorMessage))
		err = nil
	})

	if err != nil {
		return
	}

	err = d.wasmMain.wasmLinker.DefineFunc(d.wasmStore, "env", "console.log", func(messagePtr int32) {
		var message []byte
		message, err = d.mallocAssemblyScriptBytes(messagePtr)
		printf(string(message))
	})
	if err != nil {
		return
	}

	err = d.wasmMain.wasmLinker.DefineFunc(d.wasmStore, "env", "seed", func() float64 {
		return rand.Float64() * float64(time.Now().UnixMilli())
	})
	if err != nil {
		return
	}

	d.wasmInstance, err = d.wasmMain.wasmLinker.Instantiate(d.wasmStore, d.wasmMain.wasmModule)
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
	d.variableForUserFunc = d.wasmInstance.GetExport(d.wasmStore, "variableForUserPreallocated").Func()

	// bind exported internal functions
	d.__newFunc = d.wasmInstance.GetExport(d.wasmStore, "__new").Func()
	d.__pinFunc = d.wasmInstance.GetExport(d.wasmStore, "__pin").Func()
	d.__unpinFunc = d.wasmInstance.GetExport(d.wasmStore, "__unpin").Func()
	d.__collectFunc = d.wasmInstance.GetExport(d.wasmStore, "__collect").Func()

	boolType := d.wasmInstance.GetExport(d.wasmStore, "VariableType.Boolean").Global().Get(d.wasmStore).I32()
	stringType := d.wasmInstance.GetExport(d.wasmStore, "VariableType.String").Global().Get(d.wasmStore).I32()
	numberType := d.wasmInstance.GetExport(d.wasmStore, "VariableType.Number").Global().Get(d.wasmStore).I32()
	jsonType := d.wasmInstance.GetExport(d.wasmStore, "VariableType.JSON").Global().Get(d.wasmStore).I32()

	d.VariableTypeCodes = VariableTypeCodes{
		Boolean: VariableTypeCode(boolType),
		String:  VariableTypeCode(stringType),
		Number:  VariableTypeCode(numberType),
		JSON:    VariableTypeCode(jsonType),
	}

	d.allocatedMemPool = make([][]int32, d.options.MaxMemoryAllocationBuckets)
	
	// preallocate "buckets" of memory to write data buffers of different lengths to
	// allocate 2^5 bytes to 2^(5+MaxMemoryAllocationBuckets) bytes
	for i := memoryBucketOffset; i < d.options.MaxMemoryAllocationBuckets+memoryBucketOffset; i++ {
		index := i - memoryBucketOffset
		size := 1 << i

		d.allocatedMemPool[index] = make([]int32, 2)
		ptr1, err := d.allocMemForString(int32(size))

		if err != nil {
			return err
		}

		ptr2, err := d.allocMemForString(int32(size))

		if err != nil {
			return err
		}

		// currently we know there can only be two strings allocated at a time in the VariableForUser method
		// which is the only method using this pool. Knowing that, preallocate two blocks for each size bucket
		// We can then use both blocks of a particular bucket in case of a size collision between the two strings
		d.allocatedMemPool[index][0] = ptr1
		d.allocatedMemPool[index][1] = ptr2
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
	err = d.SetPlatformData(string(platformJSON))

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
	d.sdkKeyAddr = addr
	return
}

func (d *DevCycleLocalBucketing) initEventQueue(options []byte) (err error) {
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
	result, err := d.mallocAssemblyScriptBytes(addrResult.(int32))
	if err != nil {
		return
	}
	err = json.Unmarshal(result, &payload)
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

	payloadIdAddr, err := d.newAssemblyScriptString([]byte(payloadId))
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
	eventAddr, err := d.newAssemblyScriptString([]byte(event))
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

	payloadIdAddr, err := d.newAssemblyScriptString([]byte(payloadId))
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
	userAddr, err := d.newAssemblyScriptString([]byte(user))
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
	rawConfig, err := d.mallocAssemblyScriptBytes(configPtr.(int32))
	if err != nil {
		return
	}
	err = json.Unmarshal(rawConfig, &ret)
	return ret, err
}

func (d *DevCycleLocalBucketing) VariableForUser(user []byte, key string, variableType VariableTypeCode) (ret Variable, err error) {
	d.wasmMutex.Lock()
	errorMessage = ""
	defer d.wasmMutex.Unlock()

	keyAddr, preAllocatedKey, err := d.newAssemblyScriptStringWithPool([]byte(key), 0)

	if err != nil {
		return
	}

	defer func() {
		if !preAllocatedKey {
			err := d.assemblyScriptUnpin(keyAddr)
			if err != nil {
				errorf(err.Error())
			}
		}
	}()

	userAddr, preAllocatedUser, err := d.newAssemblyScriptStringWithPool(user, 1)

	if err != nil {
		return
	}

	defer func() {
		if !preAllocatedUser {
			err := d.assemblyScriptUnpin(userAddr)
			if err != nil {
				errorf(err.Error())
			}
		}
	}()

	varPtr, err := d.variableForUserFunc.Call(d.wasmStore, d.sdkKeyAddr, userAddr, len(user), keyAddr, len(key), int32(variableType), 1)
	if err != nil {
		return
	}

	var intPtr = varPtr.(int32)

	if intPtr == 0 {
		ret = Variable{}
		return
	}

	if errorMessage != "" {
		err = fmt.Errorf(errorMessage)
		return
	}
	rawVar, err := d.mallocAssemblyScriptBytes(intPtr)
	if err != nil {
		return
	}
	err = json.Unmarshal(rawVar, &ret)
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

	configAddr, err := d.newAssemblyScriptString([]byte(config))
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

	configAddr, err := d.newAssemblyScriptString([]byte(platformData))
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

	customDataAddr, err := d.newAssemblyScriptString([]byte(customData))
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
func (d *DevCycleLocalBucketing) newAssemblyScriptString(param []byte) (int32, error) {
	const objectIdString int32 = 2

	// malloc
	ptr, err := d.__newFunc.Call(d.wasmStore, int32(len(param)*2), objectIdString)
	if err != nil {
		return -1, err
	}
	addr := ptr.(int32)
	data := d.wasmMemory.UnsafeData(d.wasmStore)

	for i, c := range param {
		data[addr+int32(i*2)] = c
	}

	dataAddress := ptr.(int32)
	if dataAddress == 0 {
		return -1, errorf("Failed to allocate memory for string")
	}
	return ptr.(int32), nil
}

func (d *DevCycleLocalBucketing) newAssemblyScriptStringWithPool(param []byte, preAllocIndex int32) (int32, bool, error) {
	ptr, preAllocated, err := d.allocMemForStringPool(int32(len(param)*2), preAllocIndex)

	if err != nil {
		return -1, false, err
	}

	data := d.wasmMemory.UnsafeData(d.wasmStore)
	for i, c := range param {
		data[ptr+int32(i*2)] = c
	}

	if ptr == 0 {
		return -1, false, errorf("Failed to allocate memory for string")
	}

	return ptr, preAllocated, nil
}

func (d *DevCycleLocalBucketing) allocMemForStringPool(size int32, preAllocIndex int32) (addr int32, preAllocated bool, err error) {
	if len(d.allocatedMemPool) == 0 {
		// dont use the pool, fall through to alloc below
	} else {
		// index is the highest power value of 2 for the size we want, offset by the start of the allocation sizes
		cachedIdx := int32(math.Max(memoryBucketOffset, math.Ceil(math.Log2(float64(size))))) - memoryBucketOffset
		// if this index exceeds the max size of the pool, we'll just allocate the memory temporarily
		if cachedIdx >= int32(len(d.allocatedMemPool)) {
			warnf("String size exceeds max memory pool size, allocating new temporary block")
		} else {
			return d.allocatedMemPool[cachedIdx][preAllocIndex], true, nil
		}
	}

	// malloc
	ptr, err := d.allocMemForString(size)

	return ptr, false, err
}

func (d *DevCycleLocalBucketing) allocMemForString(size int32) (addr int32, err error) {
	const objectIdString int32 = 2

	// malloc
	ptr, err := d.__newFunc.Call(d.wasmStore, size, objectIdString)
	if err != nil {
		return -1, err
	}

	if err := d.assemblyScriptPin(ptr.(int32)); err != nil {
		return -1, err
	}

	return ptr.(int32), nil
}

// https://www.assemblyscript.org/runtime.html#memory-layout
// This skips every other index in the resulting array because
// there isn't a great way to parse UTF-16 cleanly that matches the WTF-16 format that ASC uses.
func (d *DevCycleLocalBucketing) mallocAssemblyScriptBytes(pointer int32) ([]byte, error) {
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
