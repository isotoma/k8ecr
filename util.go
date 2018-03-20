package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
)

func getContext() string {
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	context := strings.TrimSpace(string(output))
	return context
}

func getProfile() string {
	context := getContext()
	config := readConfig()
	return config[context]
}

func createSession() *session.Session {
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile:           getProfile(),
		SharedConfigState: session.SharedConfigEnable})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return sess
}
