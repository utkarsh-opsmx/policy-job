package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)


type ReleasePayload struct {
	JetId							string `json:"jetId"`
	SealId 							string `json:"sealId"`
	Branch							string `json:"branch"`
	ArtifactCreateDate 				int `json:"artifactCreateDate"`
}

type ReleaseResponse struct {
	JetId							string `json:"jetId"`
	ReleaseReady					bool `json:"releaseReady"`
	JetConsoleUrl					string `json:"jetConsoleUrl"`
	ReleaseReadyMessage				[]string `json:"releaseReadyMessage"`
	Regulations						[]Regulation `json:"regulations"`
}

type ServiceNowResponse struct {
	state 							string `json:"state"`
	startTime 						string `json:"startTime"`
	endTime							string `json:"endTime"`
	mainConfigurationItem			MainConfigurationItem `json:"mainConfigurationItem"`
}

type MainConfigurationItem struct {
	name 							string `json:"name"`
	number 							ConfigurationItemNumber `json:"number"`
}

type ConfigurationItemNumber struct {
	identifier						string `json:"identifier"`
}

type Regulation struct {
	RegulationId					string `json:"regulationId"`
	EffectiveDate					int `json:"effectiveDate"`
	EnforcementDate					int `json:"enforcementDate"`
}

const (
	opsmxToken = "X-OpsMx-Auth"
	timeout    = 600
)

type Result struct {
	response string
	err    error
}

func RunPresync(ctx context.Context) error {
	if err := validateInput(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wgErrorChan := make(chan bool)
	wgDoneChan := make(chan bool)

	for _, payload := range payloads {
		wg.Add(1)
		go func(payload string) {
			defer wg.Done()
			jobPayload := JobPayload{}
			if err := json.Unmarshal([]byte(payload), &jobPayload); err != nil {
				log.Printf("error while parsing job payload %v", err)
				wgErrorChan <- true
			}

			if(strings.TrimSpace(releaseCheckUrl) != ""){
				startValidationSteward(ctx, releaseCheckUrl, jobPayload, wgErrorChan)
			}

		}(payload)
	}

	if(strings.TrimSpace(servicenowCheckUrl) != "") {
		go startServiceNowSteward(ctx, servicenowCheckUrl, wgErrorChan)
	}

	go func() {
		wg.Wait()
		wgDoneChan <- true
	}()

	areThereAnyErrors := false
	
	for {
		select {
		case <-wgErrorChan:
			areThereAnyErrors = true
		case <-wgDoneChan:
			if areThereAnyErrors {
				return fmt.Errorf("")
			}
			return nil
		}
	}
}

func makeReleasePayload(payload JobPayload) (ReleasePayload, error) {
	epoch, err := time.Parse(time.RFC3339,payload.ArtifactCreateDate)
	if err != nil {
		return ReleasePayload{}, err
	}
	releasePayload := ReleasePayload{
		JetId: 					payload.JetId,
		SealId: 				payload.SealId,
		Branch:					gitBranch,
		ArtifactCreateDate: 	int(epoch.Unix()),
	}
	return releasePayload, nil
}

// func extractJetId(gitCommitMessage string) string {
// 	return gitCommitMessage
// }

// func extractSealId(gitCommitMessage string) string {
// 	return gitCommitMessage
// }

func extractSnowId(gitCommitMessage string) string {
	return gitCommitMessage
}

func startValidationSteward(ctx context.Context, url string, payload JobPayload, wgErrorChan chan<- bool) {
	resultChan := make(chan Result)
	defer close(resultChan)

	releasePayload, err := makeReleasePayload(payload)
	if err != nil {
		wgErrorChan <- true
		return
	}

	go releaseReadyValidation(url, resultChan, releasePayload)

	for {
		select {
		case <-ctx.Done():
			log.Printf("ERROR: Timed out/cancelled for %v", releasePayload)
			wgErrorChan <- true
			return
		case result := <-resultChan:
			if result.err != nil {
				wgErrorChan <- true
			} else {
				var releaseResponse ReleaseResponse
				if err := json.Unmarshal([]byte(result.response), &releaseResponse); err != nil {
					log.Printf("ERROR: While parsing release validation response: %v", err)
					wgErrorChan <- true
				} else if !releaseResponse.ReleaseReady {
					log.Printf("FAILURE: Release check validation failed for JetId: %s and Image: %s", payload.JetId, payload.ArtifactName)
					wgErrorChan <- true
				} else {
					log.Printf("SUCCESS: Release check validation passed for JetId: %s and Image: %s", payload.JetId, payload.ArtifactName)
				}
			}
			return
		}
	}
}

func startServiceNowSteward(ctx context.Context, url string, wgErrorChan chan<- bool) {
	resultChan := make(chan Result)
	defer close(resultChan)

	snowId := extractSnowId(gitCommitMessage)

	go serviceNowValidation(url, resultChan, snowId)

	for {
		select {
		case <-ctx.Done():
			log.Printf("ERROR: Timed out/cancelled for %s", snowId)
			wgErrorChan <- true
			return
		case result := <-resultChan:
			if result.err != nil {
				wgErrorChan <- true
			} else {
				var serviceNowResponse ServiceNowResponse
				if err := json.Unmarshal([]byte(result.response), &serviceNowResponse); err != nil {
					log.Printf("ERROR: While parsing service now response: %v", err)
					wgErrorChan <- true
				} else if !checkServiceNowStatus(serviceNowResponse) {
					log.Printf("FAILURE: Service now validation failed for SnowId: %s", snowId)
					wgErrorChan <- true
				} else {
					log.Printf("SUCCESS: Service now validation passed for SnowId: %s", snowId)
				}
			}
			return
		}
	}
}

func checkServiceNowStatus(serviceNowResponse ServiceNowResponse) bool {
	if serviceNowResponse.state == "Implement" {
		return true
	}
	if serviceNowResponse.state != "Scheduled" {
		return false
	}
	endTime, err := time.Parse(time.RFC3339, serviceNowResponse.endTime)
	if err != nil {
		log.Printf("error parsing endTime")
		return false
	}
	startTime, err := time.Parse(time.RFC3339, serviceNowResponse.startTime)
	if err != nil {
		log.Printf("error parsing startTime")
		return false
	}
	if (time.Now().Unix() > endTime.Unix()) || (time.Now().Unix() < startTime.Unix()) {
		return false
	}
	sealIdFromResponse, deploymentIdFromResponse := parseIdentifierField(serviceNowResponse)
	if sealIdFromResponse != sealId {
		return false
	}
	if deploymentIdFromResponse != deploymentId {
		return false
	}
	return true
}

func parseIdentifierField(serviceNowResponse ServiceNowResponse) (string, string) {
	identifier := serviceNowResponse.mainConfigurationItem.number.identifier
	ix := strings.LastIndex(identifier, ":")
	if ix != -1 {
		return identifier[:ix], identifier[ix+1:]
	}
	return "", ""
}

func releaseReadyValidation(url string, resultChan chan<- Result, payload ReleasePayload) {
	statusCode, responseBytes , err := getForReleaseCheckHost(httpClient, url, token, payload.JetId, payload.Branch, payload.SealId, payload.ArtifactCreateDate)
	if err != nil {
		err = fmt.Errorf("ERROR: While release validation for JetId: %s and Image: - err: %v", payload.JetId, err)
		resultChan <- Result{err: err}
		return
	}

	if statusCode != http.StatusOK {
		err = fmt.Errorf("ERROR: While release validation for JetId: %s and Image: - err: %d", payload.JetId, statusCode)
		resultChan <- Result{err: err}
		return
	}

	resultChan <- Result{response: string(responseBytes)}
}

func serviceNowValidation(url string, resultChan chan<- Result, snowId string) {
	statusCode, responseBytes , err := getForServiceNowCheckHost(httpClient, url, token, snowId)
	if err != nil {
		err = fmt.Errorf("ERROR: While servicenow validation for SnowId %s - err: %v", snowId, err)
		resultChan <- Result{err: err}
		return
	}

	if statusCode != http.StatusOK {
		err = fmt.Errorf("ERROR: While servicenow validation for SnowId %s - httpstatus code %d", snowId, statusCode)
		resultChan <- Result{err: err}
		return
	}

	resultChan <- Result{response: string(responseBytes)}
}

func validateInput() error {
	if strings.TrimSpace(token) == "" {
		return errors.New("token flag has not been set for the policy-presync binary")
	}

	if len(payloads) == 0 {
		return errors.New("payload flag has not been set for the policy-presync binary")
	}
	return nil
}

func getForReleaseCheckHost(c *http.Client, url, token string, jetId string, gitBranch string, sealId string, artifactCreateDate int) (int, []byte, error){
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add(opsmxToken, token)

	q := request.URL.Query()
	q.Add("gitBranch", gitBranch)
	q.Add("sealId", sealId)
	q.Add("jetId", jetId)
	q.Add("artifactCreateDate", strconv.Itoa(artifactCreateDate))

	request.URL.RawQuery = q.Encode()

	resp, err := c.Do(request)
	if err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, content, nil
}

func getForServiceNowCheckHost(c *http.Client, url, token string, snowId string) (int, []byte, error){
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add(opsmxToken, token)

	q := request.URL.Query()
	q.Add("snowId", snowId)

	request.URL.RawQuery = q.Encode()

	resp, err := c.Do(request)
	if err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, content, nil
}