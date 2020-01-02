package eirinistaging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/eirini-staging/builder"
	"code.cloudfoundry.org/eirini-staging/util"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
	"github.com/pkg/errors"
)

type Responder struct {
	stagingGUID        string
	completionCallback string
	eiriniAddr         string
	client             *http.Client
}

func NewResponder(stagingGUID, completionCallback, eiriniAddr, caCert, clientCrt, clientKey string) (Responder, error) {
	client, err := util.CreateTLSHTTPClient([]util.CertPaths{
		{Crt: clientCrt, Key: clientKey, Ca: caCert},
	})
	if err != nil {
		log.Println("mTLS is not configured, falling back to non-secure client")
		client = &http.Client{}
	}

	return Responder{
		stagingGUID:        stagingGUID,
		completionCallback: completionCallback,
		eiriniAddr:         eiriniAddr,
		client:             client,
	}, nil
}

func (r Responder) RespondWithFailure(failure error) {
	log.Println(failure.Error())
	cbResponse := r.createFailureResponse(failure, r.stagingGUID, r.completionCallback)

	if completeErr := r.sendCompleteResponse(cbResponse); completeErr != nil {
		fmt.Println("Error while processing failure completion callback:", completeErr.Error())
	}
}

func (r Responder) PrepareSuccessResponse(outputLocation, buildpackCfg string) (*models.TaskCallbackResponse, error) {
	resp, err := r.createSuccessResponse(outputLocation, buildpackCfg)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (r Responder) RespondWithSuccess(resp *models.TaskCallbackResponse) error {
	return r.sendCompleteResponse(resp)
}

func (r Responder) createSuccessResponse(outputMetadataLocation string, buildpackJSON string) (*models.TaskCallbackResponse, error) {
	stagingResult, err := r.getStagingResult(outputMetadataLocation)
	if err != nil {
		return nil, err
	}

	modifier := &BuildpacksKeyModifier{CCBuildpacksJSON: buildpackJSON}
	stagingResult, err = modifier.Modify(stagingResult)
	if err != nil {
		return nil, err
	}

	result, err := json.Marshal(stagingResult)
	if err != nil {
		return nil, err
	}

	annotation := cc_messages.StagingTaskAnnotation{
		CompletionCallback: r.completionCallback,
	}

	annotationJSON, err := json.Marshal(annotation)
	if err != nil {
		return nil, err
	}

	return &models.TaskCallbackResponse{
		TaskGuid:   r.stagingGUID,
		Result:     string(result),
		Failed:     false,
		Annotation: string(annotationJSON),
	}, nil
}

func (r Responder) createFailureResponse(failure error, stagingGUID, completionCallback string) *models.TaskCallbackResponse {
	annotation := cc_messages.StagingTaskAnnotation{
		CompletionCallback: completionCallback,
	}

	annotationJSON, err := json.Marshal(annotation)
	if err != nil {
		panic(err)
	}

	return &models.TaskCallbackResponse{
		TaskGuid:      stagingGUID,
		Failed:        true,
		FailureReason: failure.Error(),
		Annotation:    string(annotationJSON),
	}
}

func (r Responder) getStagingResult(path string) (builder.StagingResult, error) {
	contents, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return builder.StagingResult{}, errors.Wrap(err, "failed to read result.json")
	}
	var stagingResult builder.StagingResult
	err = json.Unmarshal(contents, &stagingResult)
	if err != nil {
		return builder.StagingResult{}, err
	}
	return stagingResult, nil
}

func (r Responder) sendCompleteResponse(response *models.TaskCallbackResponse) error {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	uri := fmt.Sprintf("%s/stage/%s/completed", r.eiriniAddr, response.TaskGuid)

	req, err := http.NewRequest("PUT", uri, bytes.NewBuffer(responseJSON))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, err := ioutil.ReadAll(resp.Body)
		var message string
		if err == nil {
			message = string(body)
		}
		return fmt.Errorf("request not successful: status=%d taskGuid=%s %s", resp.StatusCode, response.TaskGuid, message)
	}

	return nil
}
