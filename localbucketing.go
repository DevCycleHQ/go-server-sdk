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
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/devcyclehq/go-server-sdk/v2/proto"

	"github.com/bytecodealliance/wasmtime-go/v6"
)

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

type WASMLocalBucketing struct {
	wasmMain             *WASMMain
	localBucketingClient *WASMLocalBucketingClient
	bucketingObjectPool  *BucketingPool
	sdkKey               string
}

func NewWASMLocalBucketing(sdkKey string, options *DVCOptions) (*WASMLocalBucketing, error) {
	wasmMain := &WASMMain{}
	err := wasmMain.Initialize(options)
	if err != nil {
		return nil, fmt.Errorf("error while initializing wasm: %w", err)
	}

	localBucketing := &WASMLocalBucketingClient{}
	err = localBucketing.Initialize(wasmMain, sdkKey, options)
	if err != nil {
		return nil, errorf("error while initializing local bucketing", err)
	}

	bucketingObjectPool, err := NewBucketingPool(context.TODO(), wasmMain, sdkKey, options)

	if err != nil {
		return nil, err
	}

	return &WASMLocalBucketing{
		wasmMain:             wasmMain,
		localBucketingClient: localBucketing,
		bucketingObjectPool:  bucketingObjectPool,
		sdkKey:               sdkKey,
	}, nil

}

func (lb *WASMLocalBucketing) GenerateBucketedConfigForUser(user DVCUser) (ret *BucketedUserConfig, err error) {
	return lb.localBucketingClient.GenerateBucketedConfigForUser(user)
}

func (lb *WASMLocalBucketing) SetClientCustomData(customData map[string]interface{}) error {
	customDataJSON, err := json.Marshal(customData)
	if err != nil {
		return err
	}

	err = lb.localBucketingClient.SetClientCustomData(customDataJSON)

	if err != nil {
		return err
	}

	return lb.bucketingObjectPool.SetClientCustomData(customDataJSON)
}

func (lb *WASMLocalBucketing) StoreConfig(config []byte) error {
	err := lb.localBucketingClient.StoreConfig(config)
	if err != nil {
		return err
	}

	return lb.bucketingObjectPool.StoreConfig(config)
}

func (lb *WASMLocalBucketing) Variable(user DVCUser, key string, variableType string) (variable Variable, err error) {
	variableTypeCode, err := lb.localBucketingClient.VariableTypeCodeFromType(variableType)

	if err != nil {
		return Variable{}, err
	}

	// Take all the parameters and convert them to protobuf objects
	appBuild := math.NaN()
	if user.AppBuild != "" {
		appBuild, err = strconv.ParseFloat(user.AppBuild, 64)
		if err != nil {
			appBuild = math.NaN()
		}
	}
	userPB := &proto.DVCUser_PB{
		UserId:            user.UserId,
		Email:             createNullableString(user.Email),
		Name:              createNullableString(user.Name),
		Language:          createNullableString(user.Language),
		Country:           createNullableString(user.Country),
		AppBuild:          createNullableDouble(appBuild),
		AppVersion:        createNullableString(user.AppVersion),
		DeviceModel:       createNullableString(user.DeviceModel),
		CustomData:        createNullableCustomData(user.CustomData),
		PrivateCustomData: createNullableCustomData(user.PrivateCustomData),
	}

	// package everything into the root params object
	paramsPB := proto.VariableForUserParams_PB{
		SdkKey:           lb.sdkKey,
		VariableKey:      key,
		VariableType:     proto.VariableType_PB(variableTypeCode),
		User:             userPB,
		ShouldTrackEvent: true,
	}

	// Generate the buffer
	paramsBuffer, err := paramsPB.MarshalVT()

	if err != nil {
		return Variable{}, errorf("Error marshalling protobuf object in variableForUserProtobuf: %w", err)
	}

	variablePB, err := lb.bucketingObjectPool.VariableForUser(paramsBuffer)

	if err != nil {
		_ = errorf("Error getting variable for user: %w", err)
	}

	if variablePB == nil {
		return Variable{}, nil
	}

	return Variable{
		BaseVariable: BaseVariable{
			Key:   variablePB.Key,
			Type_: variablePB.Type.String(),
			Value: variablePB.GetValue(),
		},
		DefaultValue: nil,
		IsDefaulted:  false,
	}, nil
}

func (lb *WASMLocalBucketing) Close() {
	if lb.bucketingObjectPool != nil {
		lb.bucketingObjectPool.Close()
	}
}

type WASMLocalBucketingClient struct {
	wasmStore    *wasmtime.Store
	wasmInstance *wasmtime.Instance
	wasmMemory   *wasmtime.Memory
	wasiConfig   *wasmtime.WasiConfig
	wasmLinker   *wasmtime.Linker
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

	flushEventQueueFunc                   *wasmtime.Func
	eventQueueSizeFunc                    *wasmtime.Func
	onPayloadSuccessFunc                  *wasmtime.Func
	queueEventFunc                        *wasmtime.Func
	onPayloadFailureFunc                  *wasmtime.Func
	initEventQueueFunc                    *wasmtime.Func
	queueAggregateEventFunc               *wasmtime.Func
	variableForUser_PB_Preallocated       *wasmtime.Func
	setConfigDataUTF8Func                 *wasmtime.Func
	setPlatformDataUTF8Func               *wasmtime.Func
	setClientCustomDataUTF8Func           *wasmtime.Func
	generateBucketedConfigForUserUTF8Func *wasmtime.Func

	VariableTypeCodes VariableTypeCodes

	// Holds pointers to pre-allocated blocks of memory.
	allocatedMemPool []int32
	// Ptr to reserved block for byte buffer header
	byteBufferHeader int32
	errorMessage     string
}

func (d *WASMLocalBucketingClient) Initialize(wasmMain *WASMMain, sdkKey string, options *DVCOptions) (err error) {
	d.options = options
	d.wasmMain = wasmMain

	d.wasmLinker = wasmtime.NewLinker(d.wasmMain.wasmEngine)
	err = d.wasmLinker.DefineWasi()

	if err != nil {
		return
	}
	d.wasiConfig = wasmtime.NewWasiConfig()
	d.wasiConfig.InheritEnv()
	d.wasiConfig.InheritStderr()
	d.wasiConfig.InheritStdout()

	d.wasmStore = wasmtime.NewStore(d.wasmMain.wasmEngine)
	d.wasmStore.SetWasi(d.wasiConfig)

	if err != nil {
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "Date.now", func() float64 { return float64(time.Now().UnixMilli()) })
	if err != nil {
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "abort", func(messagePtr, filenamePointer, lineNum, colNum int32) {

		messagePtrData, err := d.readAssemblyScriptStringBytes(messagePtr)
		if err != nil {
			_ = errorf("Failed to read abort function parameter values - WASM Error: %s", err)
			return
		}
		filenamePointerData, err := d.readAssemblyScriptStringBytes(filenamePointer)
		if err != nil {
			_ = errorf("Failed to read abort function parameter values - WASM Error: %s", err)
			return
		}
		d.errorMessage = fmt.Sprintf("WASM Error: %s at %s:%d:%d", string(messagePtrData), string(filenamePointerData), lineNum, colNum)
		_ = errorf("WASM Error: %s", d.errorMessage)
		err = nil
	})

	if err != nil {
		return
	}

	err = d.wasmLinker.DefineFunc(d.wasmStore, "env", "console.log", func(messagePtr int32) {
		var message []byte
		message, err = d.readAssemblyScriptStringBytes(messagePtr)
		printf(string(message))
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

	d.wasmInstance, err = d.wasmLinker.Instantiate(d.wasmStore, d.wasmMain.wasmModule)
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
	d.queueEventFunc = d.wasmInstance.GetExport(d.wasmStore, "queueEvent").Func()
	d.queueAggregateEventFunc = d.wasmInstance.GetExport(d.wasmStore, "queueAggregateEvent").Func()
	d.variableForUser_PB_Preallocated = d.wasmInstance.GetExport(d.wasmStore, "variableForUser_PB_Preallocated").Func()
	d.setConfigDataUTF8Func = d.wasmInstance.GetExport(d.wasmStore, "setConfigDataUTF8").Func()
	d.setPlatformDataUTF8Func = d.wasmInstance.GetExport(d.wasmStore, "setPlatformDataUTF8").Func()
	d.setClientCustomDataUTF8Func = d.wasmInstance.GetExport(d.wasmStore, "setClientCustomDataUTF8").Func()
	d.generateBucketedConfigForUserUTF8Func = d.wasmInstance.GetExport(d.wasmStore, "generateBucketedConfigForUserUTF8").Func()

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

func (c *WASMLocalBucketingClient) VariableTypeCodeFromType(varType string) (varTypeCode VariableTypeCode, err error) {
	switch varType {
	case "Boolean":
		return c.VariableTypeCodes.Boolean, nil
	case "Number":
		return c.VariableTypeCodes.Number, nil
	case "String":
		return c.VariableTypeCodes.String, nil
	case "JSON":
		return c.VariableTypeCodes.JSON, nil
	}

	return 0, errorf("variable type %s is not a valid type", varType)
}

func (d *WASMLocalBucketingClient) setSDKKey(sdkKey string) (err error) {
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

func (d *WASMLocalBucketingClient) initEventQueue(options []byte) (err error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	optionsAddr, err := d.newAssemblyScriptString(options)
	if err != nil {
		return
	}

	_, err = d.initEventQueueFunc.Call(d.wasmStore, d.sdkKeyAddr, optionsAddr)
	err = d.handleWASMErrors("initEventQueue", err)
	return
}

func (d *WASMLocalBucketingClient) startFlushEvents() {
	d.flushMutex.Lock()
}

func (d *WASMLocalBucketingClient) finishFlushEvents() {
	d.flushMutex.Unlock()
}
func (d *WASMLocalBucketingClient) flushEventQueue() (payload []FlushPayload, err error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	addrResult, err := d.flushEventQueueFunc.Call(d.wasmStore, d.sdkKeyAddr)
	err = d.handleWASMErrors("flushEventQueue", err)
	if err != nil {
		return
	}
	result, err := d.readAssemblyScriptStringBytes(addrResult.(int32))
	if err != nil {
		return
	}
	err = json.Unmarshal(result, &payload)
	return
}

func (d *WASMLocalBucketingClient) checkEventQueueSize() (length int, err error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	result, err := d.eventQueueSizeFunc.Call(d.wasmStore, d.sdkKeyAddr)
	err = d.handleWASMErrors("eventQueueSize", err)
	if err != nil {
		return
	}
	queueLen := result.(int32)
	return int(queueLen), nil
}

func (d *WASMLocalBucketingClient) OnPayloadSuccess(payloadId string) (err error) {
	d.wasmMutex.Lock()
	defer d.wasmMutex.Unlock()

	return d.onPayloadSuccess(payloadId)
}

func (d *WASMLocalBucketingClient) onPayloadSuccess(payloadId string) (err error) {
	d.errorMessage = ""
	payloadIdAddr, err := d.newAssemblyScriptString([]byte(payloadId))
	if err != nil {
		return
	}

	_, err = d.onPayloadSuccessFunc.Call(d.wasmStore, d.sdkKeyAddr, payloadIdAddr)
	err = d.handleWASMErrors("onPayloadSuccess", err)
	return
}

func (d *WASMLocalBucketingClient) queueEvent(user, event string) (err error) {
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
			_ = errorf(err.Error())
		}
	}()
	eventAddr, err := d.newAssemblyScriptString([]byte(event))
	if err != nil {
		return
	}

	_, err = d.queueEventFunc.Call(d.wasmStore, d.sdkKeyAddr, userAddr, eventAddr)
	err = d.handleWASMErrors("queueEvent", err)
	return
}

func (d *WASMLocalBucketingClient) queueAggregateEvent(event string, config BucketedUserConfig) (err error) {
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
			_ = errorf(err.Error())
		}
	}()
	eventAddr, err := d.newAssemblyScriptString([]byte(event))
	if err != nil {
		return
	}

	_, err = d.queueAggregateEventFunc.Call(d.wasmStore, d.sdkKeyAddr, eventAddr, variationMapAddr)
	err = d.handleWASMErrors("queueAggregateEvent", err)
	return
}

func (d *WASMLocalBucketingClient) OnPayloadFailure(payloadId string, retryable bool) (err error) {
	d.wasmMutex.Lock()
	defer d.wasmMutex.Unlock()

	return d.onPayloadFailure(payloadId, retryable)
}

func (d *WASMLocalBucketingClient) onPayloadFailure(payloadId string, retryable bool) (err error) {
	d.errorMessage = ""

	payloadIdAddr, err := d.newAssemblyScriptString([]byte(payloadId))
	if err != nil {
		return
	}

	if err != nil {
		return
	}

	if retryable {
		_, err = d.onPayloadFailureFunc.Call(d.wasmStore, d.sdkKeyAddr, payloadIdAddr, 1)
		err = d.handleWASMErrors("onPayloadFailure", err)
	} else {
		_, err = d.onPayloadFailureFunc.Call(d.wasmStore, d.sdkKeyAddr, payloadIdAddr, 0)
		err = d.handleWASMErrors("onPayloadFailure", err)
	}
	return
}

func (d *WASMLocalBucketingClient) GenerateBucketedConfigForUser(user DVCUser) (ret *BucketedUserConfig, err error) {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()
	userAddr, err := d.newAssemblyScriptNoPoolByteArray(userJSON)
	if err != nil {
		return
	}

	configPtr, err := d.generateBucketedConfigForUserUTF8Func.Call(d.wasmStore, d.sdkKeyAddr, userAddr)
	err = d.handleWASMErrors("generateBucketedConfigUTF8", err)
	if err != nil {
		return
	}

	rawConfig, err := d.readAssemblyScriptByteArray(configPtr.(int32))
	if err != nil {
		return
	}
	var config BucketedUserConfig
	err = json.Unmarshal(rawConfig, &config)
	return &config, err
}

/*
 * This is a helper function to call the variableForUserPB function in the WASM module.
 * It takes a serialized protobuf message as input and returns a serialized protobuf message as output.
 */
func (d *WASMLocalBucketingClient) VariableForUser_PB(serializedParams []byte) (*proto.SDKVariable_PB, error) {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	paramsAddr, err := d.newAssemblyScriptByteArray(serializedParams)

	if err != nil {
		return nil, errorf("Error allocating WASM string: %w", err)
	}

	varPtr, err := d.variableForUser_PB_Preallocated.Call(d.wasmStore, paramsAddr, int32(len(serializedParams)))

	err = d.handleWASMErrors("variableForUserPB", err)

	if err != nil {
		return nil, err
	}

	var intPtr = varPtr.(int32)

	if intPtr == 0 {
		return nil, nil
	}

	rawVar, err := d.readAssemblyScriptByteArray(intPtr)
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

func (d *WASMLocalBucketingClient) StoreConfig(config []byte) error {
	defer func() {
		if err := recover(); err != nil {
			_ = errorf("Failed to process config: ", err)
		}
	}()
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	configParam, err := d.newAssemblyScriptNoPoolByteArray(config)
	if err != nil {
		return err
	}

	_, err = d.setConfigDataUTF8Func.Call(d.wasmStore, d.sdkKeyAddr, configParam)
	err = d.handleWASMErrors("setConfigDataUTF8", err)

	return err
}

func (d *WASMLocalBucketingClient) SetPlatformData(platformData []byte) error {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	dataAddr, err := d.newAssemblyScriptNoPoolByteArray(platformData)
	if err != nil {
		return err
	}

	_, err = d.setPlatformDataUTF8Func.Call(d.wasmStore, dataAddr)
	err = d.handleWASMErrors("setPlatformDataUTF", err)
	return err
}

func (d *WASMLocalBucketingClient) SetClientCustomData(customData []byte) error {
	d.wasmMutex.Lock()
	d.errorMessage = ""
	defer d.wasmMutex.Unlock()

	customDataAddr, err := d.newAssemblyScriptNoPoolByteArray(customData)
	if err != nil {
		return err
	}

	_, err = d.setClientCustomDataUTF8Func.Call(d.wasmStore, d.sdkKeyAddr, customDataAddr)
	err = d.handleWASMErrors("setClientCustomData", err)
	return err
}

func (d *WASMLocalBucketingClient) HandleFlushResults(result *FlushResult) {
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
func (d *WASMLocalBucketingClient) newAssemblyScriptString(param []byte) (int32, error) {
	const objectIdString int32 = 2

	// malloc
	ptr, err := d.__newFunc.Call(d.wasmStore, int32(len(param)*2), objectIdString)
	err = d.handleWASMErrors("__new (newAssemblyScriptString)", err)
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

func (d *WASMLocalBucketingClient) newAssemblyScriptNoPoolByteArray(param []byte) (int32, error) {
	const objectIdByteArray int32 = 1
	var align int32 = 0

	length := int32(len(param))

	headerPtr, err := d.__newFunc.Call(d.wasmStore, 12, 9)
	if err != nil {
		return -1, err
	}
	headerAddr := headerPtr.(int32)

	pinnedAddr, err := d.__pinFunc.Call(d.wasmStore, headerAddr)
	if err != nil {
		return -1, err
	}
	defer d.__unpinFunc.Call(d.wasmStore, pinnedAddr.(int32))

	buffer, err := d.allocMemForBuffer(length, objectIdByteArray, false)
	littleEndianBufferAddress := bytes.NewBuffer([]byte{})

	err = binary.Write(littleEndianBufferAddress, binary.LittleEndian, buffer)
	if err != nil {
		return 0, err
	}

	data := d.wasmMemory.UnsafeData(d.wasmStore)

	// Write to the first 8 bytes of the header
	for i, c := range littleEndianBufferAddress.Bytes() {
		data[headerAddr+int32(i)] = c
		data[headerAddr+int32(i)+4] = c
	}

	// Create another binary buffer to write the length of the buffer
	lengthBuffer := bytes.NewBuffer([]byte{})
	err = binary.Write(lengthBuffer, binary.LittleEndian, length<<align)
	if err != nil {
		return 0, err
	}
	// Write the length to the last 4 bytes of the header
	for i, c := range lengthBuffer.Bytes() {
		data[headerAddr+8+int32(i)] = c
	}

	// Write the buffer itself into WASM.
	for i, c := range param {
		data[buffer+int32(i)] = c
	}
	return headerAddr, err
}

func (d *WASMLocalBucketingClient) allocMemForBufferPool(size int32) (addr int32, err error) {
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

func (d *WASMLocalBucketingClient) allocMemForBuffer(size int32, classId int32, shouldPin bool) (addr int32, err error) {
	// malloc
	ptr, err := d.__newFunc.Call(d.wasmStore, size, classId)
	err = d.handleWASMErrors("__new (allocMemForBuffer)", err)
	if err != nil {
		return -1, err
	}
	if shouldPin {
		if err := d.assemblyScriptPin(ptr.(int32)); err != nil {
			return -1, err
		}
	}
	return ptr.(int32), nil
}

func (d *WASMLocalBucketingClient) newAssemblyScriptByteArray(param []byte) (int32, error) {
	const align = 0
	length := int32(len(param))

	buffer, err := d.allocMemForBufferPool(length << align)
	// Allocate the full buffer of our data - this is a buffer
	if err != nil {
		return -1, err
	}

	// Create a binary buffer to write little endian format
	littleEndianBufferAddress := bytes.NewBuffer([]byte{})

	err = binary.Write(littleEndianBufferAddress, binary.LittleEndian, buffer)
	if err != nil {
		return 0, err
	}

	data := d.wasmMemory.UnsafeData(d.wasmStore)

	// Write to the first 8 bytes of the header
	for i, c := range littleEndianBufferAddress.Bytes() {
		data[d.byteBufferHeader+int32(i)] = c
		data[d.byteBufferHeader+int32(i)+4] = c
	}

	// Create another binary buffer to write the length of the buffer
	lengthBuffer := bytes.NewBuffer([]byte{})
	err = binary.Write(lengthBuffer, binary.LittleEndian, length<<align)
	if err != nil {
		return 0, err
	}
	// Write the length to the last 4 bytes of the header
	for i, c := range lengthBuffer.Bytes() {
		data[d.byteBufferHeader+8+int32(i)] = c
	}

	// Write the buffer itself into WASM.
	for i, c := range param {
		data[buffer+int32(i)] = c
	}

	// Return the header address - as that's what's consumed on the WASM side.
	return d.byteBufferHeader, nil
}

// https://www.assemblyscript.org/runtime.html#memory-layout
// This skips every other index in the resulting array because
// there isn't a great way to parse UTF-16 cleanly that matches the WTF-16 format that ASC uses.
func (d *WASMLocalBucketingClient) readAssemblyScriptStringBytes(pointer int32) ([]byte, error) {
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

func (d *WASMLocalBucketingClient) readAssemblyScriptByteArray(pointer int32) ([]byte, error) {
	if pointer == 0 {
		return nil, errorf("null pointer passed to mallocAssemblyScriptString - cannot write string")
	}

	data := d.wasmMemory.UnsafeData(d.wasmStore)
	dataLength := byteArrayToInt(data[pointer+8 : pointer+12])

	dataPointer := byteArrayToInt(data[pointer : pointer+4])

	ret := make([]byte, dataLength)

	for i := 0; i < int(dataLength); i++ {
		ret[i] = data[int32(dataPointer)+int32(i)]
	}

	return ret, nil
}
func (d *WASMLocalBucketingClient) assemblyScriptPin(pointer int32) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptPin - cannot pin")
	}
	_, err = d.__pinFunc.Call(d.wasmStore, pointer)
	err = d.handleWASMErrors("__pin", err)
	return
}

func (d *WASMLocalBucketingClient) assemblyScriptCollect() (err error) {
	_, err = d.__collectFunc.Call(d.wasmStore)
	err = d.handleWASMErrors("__collect", err)
	return
}

func (d *WASMLocalBucketingClient) assemblyScriptUnpin(pointer int32) (err error) {
	if pointer == 0 {
		return errorf("null pointer passed to assemblyScriptUnpin - cannot unpin")
	}

	_, err = d.__unpinFunc.Call(d.wasmStore, pointer)
	err = d.handleWASMErrors("__unpin", err)
	return
}

func (d *WASMLocalBucketingClient) handleWASMErrors(prefix string, err error) error {
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
