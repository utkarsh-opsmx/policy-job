package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"bytes"
	"net/http"
	"strings"
	"sync"
	"encoding/json"
)

type DeploymentPayload struct {
	eventStatus							string `json:"jetId"`
	deployTool 							string `json:"deployTool"`
	sealId								string `json:"sealId"`
	jetId 								string `json:"jetId"`
	repoName							string `json:"repoName"`
	projectName 						string `json:"projectName"`
	branch								string `json:"branch"`
	eventSubType						string `json:"eventSubType"`
	commitId 							string `json:"commitId"`
	sourceUri							string `json:"sourceUri"`
	artifactId 							string `json:"artifactId"`
	artifactLocation					string `json:"artifactLocation"`
	targetEnvironment 					string `json:"targetEnvironment"`
	initiator							string `json:"initiator"`
	extPayload 							string `json:"extPayload"`
}

func RunPostsync(ctx context.Context) error {
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

			if(strings.TrimSpace(submitDeploymentUrl) != ""){
				startSubmitDeploymentSteward(ctx, submitDeploymentUrl, jobPayload, wgErrorChan)
			}

		}(payload)
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
			log.Printf("SUCCESS: Deployment details submitted for JetId: %s","SampleCombinedJetId")
			return nil
		}
	}
}

func MakeDeploymentPayload(payload JobPayload) (string, error) {
	repoName, err := extractRepoName(repoUrl)
	if(err != nil) {
		return "", err
	}
	deploymentPayload, err := json.Marshal(DeploymentPayload{
		eventStatus: "SUCCESS",
		deployTool: "ArgoCD",
		sealId: payload.SealId,
		jetId: payload.JetId,
		repoName: repoName,
		projectName: payload.ProjectName,
		branch: gitBranch,
		commitId: gitLastCommitId,
		sourceUri: "???",
		artifactId: payload.ArtifactId,
		artifactLocation: payload.artifactLocation,
		targetEnvironment: targetEnvironment,
		initiator: "???",
	})
	return string(deploymentPayload), err
}

func startSubmitDeploymentSteward(ctx context.Context, url string, payload JobPayload, wgErrorChan chan<- bool) {
	resultChan := make(chan Result)
	defer close(resultChan)

	deploymentPayload, err := MakeDeploymentPayload(payload)
	if err != nil {
		wgErrorChan <- true
		return
	}

	go submitDeployment(url, resultChan, deploymentPayload)

	for {
		select {
		case <-ctx.Done():
			log.Printf("ERROR: Timed out/cancelled for %s", deploymentPayload)
			wgErrorChan <- true
			return
		case result := <-resultChan:
			if result.err != nil {
				wgErrorChan <- true
			}
			return
		}
	}
}

func submitDeployment(url string, resultChan chan<- Result, payload string) {
	statusCode, responseBytes , err := postToHost(httpClient, url, token, []byte(payload))
	if err != nil {
		err = fmt.Errorf("ERROR: While submitting deployment payload for JetId: %s and Image: - err: %v", payload, err)
		resultChan <- Result{err: err}
		return
	}

	if statusCode != http.StatusOK {
		err = fmt.Errorf("ERROR: While submitting deployment payload for JetId: %s and Image: - err: %d", payload, statusCode)
		resultChan <- Result{err: err}
		return
	}

	resultChan <- Result{response: string(responseBytes)}
}


func postToHost(c *http.Client, url, token string, serializeddata []byte) (int, []byte, error) {

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(serializeddata))
	if err != nil {
		return 0, nil, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add(opsmxToken, token)

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

// extractRepoName extracts the repository name from a GitHub URL.
func extractRepoName(githubURL string) (string, error) {
	// Trim the trailing slash (if any)
	githubURL = strings.TrimSuffix(githubURL, "/")

	// Split the URL by "/"
	parts := strings.Split(githubURL, "/")

	// A valid GitHub repo URL should have at least: https://github.com/user/repo
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GitHub URL")
	}

	// The last part is the repository name
	return parts[len(parts)-1], nil
}