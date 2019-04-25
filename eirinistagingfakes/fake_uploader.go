// Code generated by counterfeiter. DO NOT EDIT.
package eirinistagingfakes

import (
	"sync"

	"code.cloudfoundry.org/eirini-staging"
)

type FakeUploader struct {
	UploadStub        func(string, string) error
	uploadMutex       sync.RWMutex
	uploadArgsForCall []struct {
		arg1 string
		arg2 string
	}
	uploadReturns struct {
		result1 error
	}
	uploadReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeUploader) Upload(arg1 string, arg2 string) error {
	fake.uploadMutex.Lock()
	ret, specificReturn := fake.uploadReturnsOnCall[len(fake.uploadArgsForCall)]
	fake.uploadArgsForCall = append(fake.uploadArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("Upload", []interface{}{arg1, arg2})
	fake.uploadMutex.Unlock()
	if fake.UploadStub != nil {
		return fake.UploadStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.uploadReturns
	return fakeReturns.result1
}

func (fake *FakeUploader) UploadCallCount() int {
	fake.uploadMutex.RLock()
	defer fake.uploadMutex.RUnlock()
	return len(fake.uploadArgsForCall)
}

func (fake *FakeUploader) UploadCalls(stub func(string, string) error) {
	fake.uploadMutex.Lock()
	defer fake.uploadMutex.Unlock()
	fake.UploadStub = stub
}

func (fake *FakeUploader) UploadArgsForCall(i int) (string, string) {
	fake.uploadMutex.RLock()
	defer fake.uploadMutex.RUnlock()
	argsForCall := fake.uploadArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeUploader) UploadReturns(result1 error) {
	fake.uploadMutex.Lock()
	defer fake.uploadMutex.Unlock()
	fake.UploadStub = nil
	fake.uploadReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeUploader) UploadReturnsOnCall(i int, result1 error) {
	fake.uploadMutex.Lock()
	defer fake.uploadMutex.Unlock()
	fake.UploadStub = nil
	if fake.uploadReturnsOnCall == nil {
		fake.uploadReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.uploadReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeUploader) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.uploadMutex.RLock()
	defer fake.uploadMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeUploader) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ eirinistaging.Uploader = new(FakeUploader)
