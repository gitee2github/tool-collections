/*
Copyright 2019 The openeuler community Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"sync"
)

type SigRepoCheck struct {
	FileName string
	GiteeToken string
}



var sigRepoCheck = &SigRepoCheck{}

func SigInitRunFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&sigRepoCheck.FileName, "filename", "f", "", "the file name of sig file")
	cmd.Flags().StringVarP(&sigRepoCheck.GiteeToken, "giteetoken", "g", "", "the gitee token")
}

func buildSigCommand() *cobra.Command {
	sigCommand := &cobra.Command{
		Use:   "sig",
		Short: "operation on sigs",
	}

	checkCommand := &cobra.Command{
		Use:   "checkrepo",
		Short: "check repo legality in sig yaml",
		Run: func(cmd *cobra.Command, args []string) {
			checkError(cmd, CheckSigRepo())
		},
	}
	SigInitRunFlags(checkCommand)
	sigCommand.AddCommand(checkCommand)

	return sigCommand
}

func CheckSigRepo() error {
	var wg sync.WaitGroup
	var invalidProjects []string
	fmt.Printf("Starting to validating all of the repos in sig file %s\n", sigRepoCheck.FileName)
	if _, err := os.Stat(sigRepoCheck.FileName); os.IsNotExist(err) {
		return fmt.Errorf("sig file not existed %s", sigRepoCheck.FileName)
	}

	// Setting up gitee handler
	giteeHandler := NewGiteeHandler(sigRepoCheck.GiteeToken)
	sigChannel := make(chan string, 50)
	stopCh := SetupSignalHandler()
	resultChannel := make(chan string, 50)
	// Running 5 workers to check the repo status
	go func() {
		for rs := range resultChannel {
			invalidProjects = append(invalidProjects, rs)
		}
	}()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go giteeHandler.ValidateRepo(&wg, stopCh, resultChannel, sigChannel, "open_euler")
	}

	scanner := NewDirScanner("")
	err := scanner.ScanSigYaml(sigRepoCheck.FileName, sigChannel)
	wg.Wait()
	close(resultChannel)
	if err != nil {
		return err
	}
	if len(invalidProjects) != 0 {
		return fmt.Errorf("[Import] Failed to recognize gitee projects: %s\n", strings.Join(invalidProjects,","))
	}
	fmt.Printf("Projects successfully verified.")
	return nil
}

