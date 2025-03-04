package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"bytes"
)

func getDeploymentIdAndSealId() error {
	app := "kubectl"
	//kubectl get app <appname> -o jsonpath='{.metadata.labels}'
	arg0 := "get"
	arg1 := "app"
	arg2 := argocdAppName
	arg3 := "-o" 
	arg4 := "jsonpath='{.metadata.labels}'"
	arg6 := "-n"
	arg7 := argocdNamespace
	cmd := exec.Command(app, arg0,arg1,arg2,arg3,arg4, arg6, arg7)
	var out bytes.Buffer
var stderr bytes.Buffer
cmd.Stdout = &out
cmd.Stderr = &stderr
	err := cmd.Run()
	// labelsJson, err := cmd.Output()
	labelsJson := out.String()
	if err != nil {
		return fmt.Errorf("command %s failed with output: %s and error: %v", app, &stderr, err)
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(labelsJson), &labels); err != nil {
		return err
	}
	sealId = labels["sealId"]
	deploymentId = labels["deploymentId"]
	return nil
}

// func getImageNameToBranchMapping() (map[string]string, error) {
// 	cmd := "kubectl"
// 	//kubectl get app <appname> -o jsonpath='{.metadata.labels}'
// 	arg0 := "get app"
// 	arg1 := argocdAppName
// 	arg2 := "-o jsonpath"
// 	arg3 := "'{.spec.sources}'"
// 	sourcesJson, err := exec.Command(cmd, arg0,arg1,arg2,arg3).Output()
// 	if err != nil {
// 		return map[string]string{}, err;
// 	}
// 	var sources []map[string]string
// 	if err := json.Unmarshal(sourcesJson, &sources); err != nil {
// 		return map[string]string{}, err;
// 	}
// 	var results map[string]string
// 	for source := range sources {
// 		imageName := extractImageNamefromRepoUrl(source["repoURL"])
// 		results[imageName] = source["targetRevision"]
// 	}
// 	return results, nil
// }

// func extractImageNamefromRepoUrl() string {

// }