package eirinistaging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/eirini-staging/builder"
	"code.cloudfoundry.org/eirini-staging/util"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
	"github.com/pkg/errors"
)

type Responder struct {
	StagingGUID        string
	CompletionCallback string
	EiriniAddr         string
	CACert             string
	ClientCrt          string
	ClientKey          string
}

func NewResponder(stagingGUID string, completionCallback string, eiriniAddr string) Responder {
	return Responder{
		StagingGUID:        stagingGUID,
		CompletionCallback: completionCallback,
		EiriniAddr:         eiriniAddr,
	}
}

func (r Responder) RespondWithFailure(failure error) {
	cbResponse := r.createFailureResponse(failure, r.StagingGUID, r.CompletionCallback)

	if completeErr := r.sendCompleteResponse(cbResponse); completeErr != nil {
		fmt.Println("Error processsing completion callback:", completeErr.Error())
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
		CompletionCallback: r.CompletionCallback,
	}

	annotationJSON, err := json.Marshal(annotation)
	if err != nil {
		return nil, err
	}

	return &models.TaskCallbackResponse{
		TaskGuid:   r.StagingGUID,
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

	uri := fmt.Sprintf("%s/stage/%s/completed", r.EiriniAddr, response.TaskGuid)

	req, err := http.NewRequest("PUT", uri, bytes.NewBuffer(responseJSON))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	client, err := util.CreateTLSHTTPClient([]util.CertPaths{
		{Crt: r.ClientCrt, Key: r.ClientKey, Ca: r.CACert},
	})

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return errors.New("Request not successful")
	}

	return nil
}
