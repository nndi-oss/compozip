package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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
	Name         string `json="name=name"`
	FileLocation string
}

func vendorHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	projectURLName := string(params["name"])
	archiveFormat := strings.ToLower(params["extension"])
	if strings.Compare(archiveFormat, "zip") != 0 && strings.Compare(archiveFormat, "tar") != 0 {
		appLogger.Warn("Invalid format type specified", "extension", archiveFormat)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Please specify valid archive type, either 'zip' or 'tar'")
		return
	}
	appLogger.Info("Processing request for vendor/extension", "vendor", projectURLName, "extension", archiveFormat)
	multipartReader, err := r.MultipartReader()
	if err != nil {
		appLogger.Error("Failed to get MultipartReader.", "error", err)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "File could not be read in request")
		return
	}
	var MaxMemoryBytes int64 = 1024 * 1000
	// 1. save composer.json in new directory
	multiPartForm, err := multipartReader.ReadForm(MaxMemoryBytes)
	if err != nil {
		appLogger.Error("Failed to parse Multipart-Form.", "error", err)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprintf(w, "File could not be read in request")
		return
	}
	composerFiles := multiPartForm.File["composer"]
	if composerFiles == nil || len(composerFiles) < 1 {
		appLogger.Error("Failed to read 'composer' file from form-data.")
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "File could not be read in request")
		return
	}
	composerFile := composerFiles[0] // first file
	composerJSONBytes, err := ioutil.ReadFile(composerFile.Filename)
	if err != nil {
		appLogger.Error("Failed to read composer.json file from disk", "error", err)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "File could not be read in request.")
		return
	}
	// 2. parse the composer.json into a composerProject struct
	var composerJSON composerProject
	err = json.Unmarshal(composerJSONBytes, &composerJSON)
	if err != nil {
		appLogger.Error("Failed to parse JSON", "filename", composerFile.Filename, "error", err)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "File could not be read in request.")
		return
	}
	// 3. get the name of the project
	// w.WriteHeader(201)
	// fmt.Fprintf(w, "Got project name from composer.json: %s", composerJSON.Name)
	// 4. Create a directory for the project
	dir, err := ioutil.TempDir(uploadsDir, projectURLName)
	if err != nil {
		appLogger.Error("Failed to create tmp directory.", "error", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Failed to validate composer.json file - please submit a valid composer file")
		return
	}
	// 5. Copy composer.json to the directory
	ioutil.WriteFile(path.Join(dir, "composer.json"), composerJSONBytes, 0664)
	// 6. Run composer validate
	// var outputStream []byte
	// 6.1. If there are validation errors, return 400 Bad Request
	cmd := exec.Command("composer", "validate")
	cmd.Dir = dir
	appLogger.Debug("Running composer validate.", "PWD", dir)
	output, err := cmd.CombinedOutput()
	if err != nil || !cmd.ProcessState.Success() {
		appLogger.Debug("Failed to run 'composer validate'.", "error", err, "output", string(output))
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Failed to validate composer.json file - please submit a valid composer file")
		return
	}
	// 7. Record the project and status in the database (project -> DOWNLOADING)
	// TODO:implement step 7
	// 8. Create a "worker" to download the dependencies, return 201 Accepted
	cmd = exec.Command("composer", "install")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	appLogger.Debug("Running composer install.", "PWD", dir)
	err = cmd.Run()
	// output, err = cmd.CombinedOutput()
	if err != nil || !cmd.ProcessState.Success() {
		appLogger.Error("Failed to run 'composer install'.", "error", err) // string(output))
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprintf(w, "Failed to download Composer depedencies.") // Got output: %s", string(output))
		return
	}
	// 9. Run `composer archive` to archive the composer dependencies
	cmd = exec.Command("composer", "archive",
		"--file=vendor",
		fmt.Sprintf("--format=%s", strings.ToLower(archiveFormat)),
		"--quiet",
	)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	appLogger.Debug("Running composer archive.", "PWD", dir)
	err = cmd.Run()
	// output, err = cmd.CombinedOutput()
	if err != nil || !cmd.ProcessState.Success() {
		appLogger.Error("Failed to run 'composer archive'", "error", err) // string(output))
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Failed to create Composer archive.")
		return
	}
	composerZIPBytes, err := ioutil.ReadFile(path.Join(dir, "vendor.zip"))
	if err != nil {
		appLogger.Error("Failed to run ReadFile vendor.zip", "error", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Failed to create Composer archive.")
		return
	}

	sendDownload(w, composerZIPBytes)
	// 10. Update status of (project -> COMPLETE) in the database
	// 11. Clean up, remove archive after 10 minutes
}

func sendDownload(w http.ResponseWriter, fileBytes []byte) {
	w.Header().Add("Content-Type", "application/force-download")
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Type", "application/download")
	w.Header().Add("Content-Description", "File Transfer")
	w.Header().Add("Content-Transfer-Encoding", "binary")
	w.Header().Add("Content-Disposition", "attachment; filename=\"vendor.zip\"")
	w.Header().Add("Expires", "0")
	w.Header().Add("Cache-Control", "must-revalidate, post-check=0, pre-check=0")
	w.Header().Add("Pragma", "public")
	w.Write(fileBytes) // all stream contents will be sent to the response
}

func main() {
	flag.Parse()

	appLogger = hclog.New(&hclog.LoggerOptions{
		Name:  "compozipd",
		Level: hclog.LevelFromString("DEBUG"),
	})
	router := mux.NewRouter()
	router.HandleFunc("/vendor/{name}/{extension}", vendorHandler).Methods("POST")
	http.Handle("/", router)
	appLogger.Info("Starting server...")
	if err := http.ListenAndServe(bind, nil); err != nil {
		appLogger.Error("Failed to start Server", "error", err)
	}
}
