package main

// TODO: Add goroutine to perform clean up, remove directories after 10 minutes
// TODO: Record the project and status in a database (e.g. project: VALIDATING)

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
)

var (
	uploadsDir string
	bind       string
	appLogger  hclog.Logger
)

func init() {
	flag.StringVar(&bind, "h", ":8080", "Address to bind the server to default ':8080'")
	flag.StringVar(&uploadsDir, "u", ".", "Upload directory")
}

type composerProject struct {
	ProjectName string `json:"name"`
	Filename    string `json:"-"`
}

func parseURLParameters(w http.ResponseWriter, r *http.Request) (string, string, error) {
	params := mux.Vars(r)
	projectURLName := string(params["name"])
	archiveFormat := strings.ToLower(params["extension"])
	if strings.Compare(archiveFormat, "zip") != 0 && strings.Compare(archiveFormat, "tar") != 0 {
		appLogger.Warn("Invalid format type specified", "extension", archiveFormat)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Please specify valid archive type, either 'zip' or 'tar'")
		return projectURLName, archiveFormat, errors.New("Invalid file type specified")
	}
	return projectURLName, archiveFormat, nil
}

func readMultipartForm(w http.ResponseWriter, r *http.Request) (*multipart.FileHeader, []byte, error) {
	multipartReader, err := r.MultipartReader()
	if err != nil {
		appLogger.Error("Failed to get MultipartReader.", "error", err)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "File could not be read in request")
		return nil, nil, err
	}
	var MaxMemoryBytes int64 = 1024 * 1000
	// 1. save composer.json in new directory
	multiPartForm, err := multipartReader.ReadForm(MaxMemoryBytes)
	if err != nil {
		appLogger.Error("Failed to parse Multipart-Form.", "error", err)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprintf(w, "File could not be read in request")
		return nil, nil, err
	}
	composerFiles := multiPartForm.File["composer"]
	if composerFiles == nil || len(composerFiles) < 1 {
		appLogger.Error("Failed to read 'composer' file from form-data.")
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "File could not be read in request")
		return nil, nil, err
	}
	composerFormPart := composerFiles[0] // first file
	composerFile, err := composerFormPart.Open()
	defer composerFile.Close()

	composerJSONBytes, err := ioutil.ReadAll(composerFile)
	if err != nil {
		appLogger.Error("Failed to read composer.json file from disk", "error", err)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "File could not be read in request.")
		return nil, nil, err
	}
	return composerFormPart, composerJSONBytes, nil
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
	return &composerJSON, nil
}

func createProjectDirectory(w http.ResponseWriter, directoryName string, data []byte) (string, error) {
	// TODO: SHA256 of the data to be the directory name?
	dir, err := ioutil.TempDir(uploadsDir, directoryName)
	if err != nil {
		appLogger.Error("Failed to create tmp directory.", "error", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Failed to validate composer.json file - please submit a valid composer file")
		return dir, err
	}
	err = ioutil.WriteFile(path.Join(dir, "composer.json"), data, 0664)

	return dir, err
}

func composerValidate(w http.ResponseWriter, dir string) error {
	cmd := exec.Command("composer", "validate")
	cmd.Dir = dir
	appLogger.Debug("Running composer validate.", "PWD", dir)
	output, err := cmd.CombinedOutput()
	if err != nil || !cmd.ProcessState.Success() {
		appLogger.Debug("Failed to run 'composer validate'.", "error", err, "output", string(output))
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Failed to validate composer.json file - please submit a valid composer file")
		return err
	}
	return nil
}

func composerInstall(w http.ResponseWriter, dir string) error {
	cmd := exec.Command("composer", "install")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	appLogger.Debug("Running composer install.", "PWD", dir)
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

func composerArchive(w http.ResponseWriter, dir string, archiveFormat string) error {
	cmd := exec.Command("composer", "archive",
		"--file=vendor",
		fmt.Sprintf("--format=%s", strings.ToLower(archiveFormat)),
		"--quiet",
	)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	appLogger.Debug("Running composer archive.", "PWD", dir)
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

func sendDownload(w http.ResponseWriter, dir, archiveFormat string) error {
	vendorZIP := path.Join(dir, fmt.Sprintf("vendor.%s", archiveFormat))
	composerZIPBytes, err := ioutil.ReadFile(vendorZIP)
	if err != nil {
		appLogger.Error("Failed to run ReadFile vendor archive", "error", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Failed to create Composer archive.")
		return err
	}
	appLogger.Info("Sending vendor archive to client", "file", vendorZIP)
	w.Header().Add("Content-Type", "application/force-download")
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Type", "application/download")
	w.Header().Add("Content-Description", "File Transfer")
	w.Header().Add("Content-Transfer-Encoding", "binary")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"vendor.%s\"", archiveFormat))
	w.Header().Add("Expires", "0")
	w.Header().Add("Cache-Control", "must-revalidate, post-check=0, pre-check=0")
	w.Header().Add("Pragma", "public")
	w.Write(composerZIPBytes) // all stream contents will be sent to the response
	return nil
}

// VendorHandler Handles the Http request
func VendorHandler(w http.ResponseWriter, r *http.Request) {
	projectURLName, archiveFormat, err := parseURLParameters(w, r)
	if err != nil {
		return
	}
	appLogger.Info("Processing request for vendor/extension",
		"vendor", projectURLName, "extension", archiveFormat)
	// 2. parse the composer.json into a composerProject struct
	formFile, composerJSONBytes, err := readMultipartForm(w, r)
	if err != nil {
		return
	}
	composerJSON, err := parseComposerJSON(w, composerJSONBytes, formFile.Filename)
	if err != nil {
		return
	}
	composerJSON.Filename = projectURLName
	dir, err := createProjectDirectory(w, projectURLName, composerJSONBytes)
	if err != nil {
		return
	}
	appLogger.Info("Processing composer.json", "project", composerJSON.ProjectName, "directory", dir)

	if err = composerValidate(w, dir); err != nil {
		return
	}
	if err = composerInstall(w, dir); err != nil {
		return
	}

	if err = composerArchive(w, dir, archiveFormat); err != nil {
		return
	}

	sendDownload(w, dir, archiveFormat)
}

func phpAndComposerExist() bool {
	err := exec.Command("php", "--version && composer --version").Run()
	return err == nil
}

func main() {
	flag.Parse()

	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Specified directory '%s' does not exist or could not be found", uploadsDir)
		return
	}

	appLogger = hclog.New(&hclog.LoggerOptions{
		Name:  "compozipd",
		Level: hclog.LevelFromString("DEBUG"),
	})

	if !phpAndComposerExist() {
		fmt.Fprint(os.Stderr, `Either PHP or Composer was not found in your $PATH.
Please make sure you have both 'php' and 'composer' installed.

Download PHP: http://php.net/downloads.php
Download Composer: https://getcomposer.org

Thanks! :)`)
		return
	}

	router := mux.NewRouter()
	router.HandleFunc("/vendor/{name}/{extension}", VendorHandler).Methods("POST")
	http.Handle("/", router)
	appLogger.Info("Starting server", "address", bind, "workingDirectory", uploadsDir)
	if err := http.ListenAndServe(bind, nil); err != nil {
		appLogger.Error("Failed to start Server", "error", err)
	}
}
