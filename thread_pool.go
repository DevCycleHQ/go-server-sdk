package devcycle

import (
	_ "embed"
	"sync"
)

type WorkerJobType int

const (
	VariableForUser WorkerJobType = iota
)

var (
	//go:embed testdata/fixture_large_config.json
	thread_config string
)

type LocalBucketingWorker struct {
	localBucketing *DevCycleLocalBucketing
	configEtag     string
	configData     []byte
	workerMutex    *sync.Mutex
}

type WorkerPayload struct {
	Type_ WorkerJobType
}

type VariableForUserPayload struct {
	WorkerPayload
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
	w.workerMutex = &sync.Mutex{}
	return
}

func (w *LocalBucketingWorker) Process(payload interface{}) interface{} {
	var workerPayload = payload.(*VariableForUserPayload)
	defer w.workerMutex.Unlock()
	if workerPayload.Type_ == VariableForUser {
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
	return nil
}

func (w *LocalBucketingWorker) variableForUser(user *[]byte, key *string, variableType VariableTypeCode) (Variable, error) {
	return w.localBucketing.VariableForUser(*user, *key, variableType)
}

func (w *LocalBucketingWorker) StoreConfig(configData []byte) error {
	w.workerMutex.Lock()
	defer w.workerMutex.Unlock()
	return w.localBucketing.StoreConfig(configData)
}

func (w *LocalBucketingWorker) BlockUntilReady() {
	w.workerMutex.Lock()
}
func (w *LocalBucketingWorker) Interrupt() {}
func (w *LocalBucketingWorker) Terminate() {}
