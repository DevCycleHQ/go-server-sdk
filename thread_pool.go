package devcycle

import (
	_ "embed"
	"sync"
)

type LocalBucketingWorker struct {
	localBucketing *DevCycleLocalBucketing
	configEtag     string
	configData     []byte
	// used to hold this worker out of the pool while its state is being externally updated (e.g. a new config)
	externalBusyMutex *sync.Mutex
}

type VariableForUserPayload struct {
	User         *[]byte
	Key          *string
	VariableType VariableTypeCode
}

type VariableForUserResponse struct {
	Variable *Variable
	Err      error
}

func (w *LocalBucketingWorker) Initialize(sdkKey string, options *DVCOptions) (err error) {
	w.localBucketing = &DevCycleLocalBucketing{}
	// TODO figure out how to track events with the new threading
	options.DisableAutomaticEventLogging = true
	err = w.localBucketing.Initialize(sdkKey, options)
	w.externalBusyMutex = &sync.Mutex{}
	return
}

func (w *LocalBucketingWorker) Process(payload interface{}) interface{} {
	var variableForUserPayload = payload.(*VariableForUserPayload)

	variable, err := w.variableForUser(
		variableForUserPayload.User,
		variableForUserPayload.Key,
		variableForUserPayload.VariableType,
	)

	return VariableForUserResponse{
		Variable: &variable,
		Err:      err,
	}
}

func (w *LocalBucketingWorker) variableForUser(user *[]byte, key *string, variableType VariableTypeCode) (Variable, error) {
	return w.localBucketing.VariableForUser(*user, *key, variableType)
}

func (w *LocalBucketingWorker) StoreConfig(configData []byte) error {
	w.externalBusyMutex.Lock()
	defer w.externalBusyMutex.Unlock()
	return w.localBucketing.StoreConfig(configData)
}

func (w *LocalBucketingWorker) SetClientCustomData(customData []byte) error {
	w.externalBusyMutex.Lock()
	defer w.externalBusyMutex.Unlock()
	return w.localBucketing.SetClientCustomData(customData)
}

func (w *LocalBucketingWorker) BlockUntilReady() {
	w.externalBusyMutex.Lock()
	defer w.externalBusyMutex.Unlock()
}

func (w *LocalBucketingWorker) Interrupt() {}
func (w *LocalBucketingWorker) Terminate() {}
