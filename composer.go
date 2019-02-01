package main

// Composer commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
	dummyComposer = `{
	"name": "compozip/generated",
	"description": "This is a stub composer.json generated because you uploaded a composer.lock file. Please discard it and use your original composer.json.",
	"license": "MIT",
	"require": {
		"php":">=5.6.30"
	}
}`
)

type composerProject struct {
	ProjectName    string `json:"name"`
	ContentHash    string `json:"content-hash,omitempty"`
	directory      string
	isComposerLock bool
}

func (c *composerProject) getName() string {
	if c.isComposerLock {
		return c.ContentHash
	}
	return c.ProjectName
}

func parseComposerJSON(w http.ResponseWriter, composerJSONBytes []byte, filename string) (*composerProject, error) {
	var composerJSON composerProject
	err := json.Unmarshal(composerJSONBytes, &composerJSON)
	if err != nil {
		appLogger.Error("Failed to parse JSON", "filename", filename, "error", err)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "File could not be read in request.")
		return nil, err
	}

	composerJSON.isComposerLock = strings.HasSuffix(filename, ".lock")
	return &composerJSON, nil
}

func phpAndComposerExist() bool {
	err := exec.Command("php", "--version && composer --version").Run()
	return err == nil
}

func composerValidate(w http.ResponseWriter, composerJSON *composerProject) error {
	cmd := exec.Command("composer", "validate")
	cmd.Dir = composerJSON.directory
	appLogger.Debug("Running composer validate.", "PWD", composerJSON.directory)
	output, err := cmd.CombinedOutput()
	if err != nil || !cmd.ProcessState.Success() {
		appLogger.Debug("Failed to run 'composer validate'.", "error", err, "output", string(output))
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Failed to validate Composer file - please submit a valid composer.json or composer.lock file")
		return err
	}
	return nil
}

func composerInstall(w http.ResponseWriter, composerJSON *composerProject) error {
	cmd := exec.Command("composer", "install")
	cmd.Dir = composerJSON.directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	appLogger.Debug("Running composer install.", "PWD", composerJSON.directory)
	err := cmd.Run()
	// output, err = cmd.CombinedOutput()
	if err != nil || !cmd.ProcessState.Success() {
		appLogger.Error("Failed to run 'composer install'.", "error", err) // string(output))
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprintf(w, "Failed to download Composer depedencies.") // Got output: %s", string(output))
		return err
	}
	return nil
}

func composerArchive(w http.ResponseWriter, composerJSON *composerProject, archiveFormat string) error {

	if composerJSON.isComposerLock {
		appLogger.Debug("Including dummy composer.json in generated vendor archive'",
			"directory", composerJSON.directory)
	}

	cmd := exec.Command("composer", "archive",
		"--file=vendor",
		fmt.Sprintf("--format=%s", strings.ToLower(archiveFormat)),
		"--quiet",
	)
	cmd.Dir = composerJSON.directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	appLogger.Debug("Running composer archive.", "PWD", composerJSON.directory)
	err := cmd.Run()
	// output, err = cmd.CombinedOutput()
	if err != nil || !cmd.ProcessState.Success() {
		appLogger.Error("Failed to run 'composer archive'", "error", err) // string(output))
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Failed to create Composer archive.")
		return err
	}
	return nil
}
