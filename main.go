package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var syncType string
var releaseCheckUrl, servicenowCheckUrl, gitCommitMessage, token, repoUrl, gitBranch, gitLastCommitId, targetEnvironment string
var submitDeploymentUrl string
var sealId = "09959"
var deploymentId = "114041"

type JobPayload struct {
	OrganizationName 			string `json:"organizationName,omitempty"`
	ArtifactName  				string `json:"artifactName"`
	ArtifactTag     			string `json:"artifactTag"`
	ArtifactId					string `json:"artifactId"`
	ArtifactCreateDate			string `json:"artifactCreateDate"`
	JetId						string `json:"jetId"`
	SealId						string `json:"sealId"`
	DeploymentId				string `json:"deploymentId"`
	ProjectName					string `json:"projectName"`
	artifactLocation			string `json:"artifactLocation"`
}

var payloads = make([]string, 0)

var rootCmd = &cobra.Command{
	Use:   "policy-job",
	Short: "This is a go client for performing validating deployments in presync job via policy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if syncType == "presync" {
			//TODO: the context is cancelled with the timeout, this can be changed to with cancel without the timeout if this starts malfunctioning
			ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
			defer cancel()

			if err := RunPresync(ctx); err != nil {
				return err
			}
			return nil
		} else if syncType == "postsync" {
			//TODO: the context is cancelled with the timeout, this can be changed to with cancel without the timeout if this starts malfunctioning
			ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
			defer cancel()

			if err := RunPostsync(ctx); err != nil {
				return err
			}
			return nil
		} else {
			return fmt.Errorf("sync-type should either be presync or postsync")
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Printf("error: %v", err)
		log.Printf("FAILURE: %s", syncType)
		os.Exit(1)
	}
	log.Printf("SUCCESS: %s", syncType)
}

func init() {
	rootCmd.Flags().StringVarP(&releaseCheckUrl, "release-check-url", "u", "", "release check url")
	rootCmd.Flags().StringVarP(&servicenowCheckUrl, "servicenow-check-url", "s", "", "servicenow check url")
	rootCmd.Flags().StringVarP(&submitDeploymentUrl, "submit-deployment-url", "d", "", "submit deployment url")
	rootCmd.Flags().StringVarP(&gitBranch, "git-branch", "b", "", "git branch")
	rootCmd.Flags().StringVarP(&gitCommitMessage, "git-last-commit-message", "c", "", "git commit message")
	rootCmd.Flags().StringVarP(&token, "service-token", "t", "", "service token")
	rootCmd.Flags().StringArrayVarP(&payloads, "payload", "p", []string{}, "payload")
	rootCmd.Flags().StringVarP(&syncType, "sync-type", "y", "", "sync type")
	rootCmd.Flags().StringVarP(&repoUrl, "repo-url", "", "", "repo url")
	rootCmd.Flags().StringVarP(&gitLastCommitId, "git-last-commitId", "", "", "git last commit id")
	rootCmd.Flags().StringVarP(&targetEnvironment, "target-environment", "", "", "target environment")
	// rootCmd.Flags().StringVarP(&sealId, "sealId", "", "", "seal id from manifests")
	// rootCmd.Flags().StringVarP(&deploymentId, "deploymentId", "", "", "deployment id from manifests")
}

func main() {
	Execute()
}
