package main

// TODO: Add goroutine to perform clean up, remove directories after 10 minutes
// TODO: Record the project and status in a database (e.g. project: VALIDATING)

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	_ "github.com/nndi-oss/compozip/statik"
	"github.com/rakyll/statik/fs"
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

// VendorHandler Handles the Http request
func VendorHandler(w http.ResponseWriter, r *http.Request) {
	archiveFormat, err := parseURLParameters(w, r)
	if err != nil {
		return
	}
	formFile, composerJSONBytes, err := readMultipartForm(w, r)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Error: %s", err)
		return
	}
	composerJSON, err := parseComposerJSON(w, composerJSONBytes, formFile.Filename)
	if err != nil {
		return
	}
	appLogger.Info("Processing request for project",
		"name-id", composerJSON.getName(),
		"extension", archiveFormat)

	dir, err := createProjectDirectory(w, composerJSON, composerJSONBytes)
	if err != nil {
		return
	}
	composerJSON.directory = dir
	if composerJSON.isComposerLock {
		appLogger.Info("Processing composer.lock file",
			"hash", composerJSON.ContentHash,
			"directory", composerJSON.directory)
	} else {
		appLogger.Info("Processing composer.json file",
			"project", composerJSON.ProjectName,
			"directory", composerJSON.directory)
	}

	if !composerJSON.isComposerLock {
		if err = composerValidate(w, composerJSON); err != nil {
			return
		}
	}

	if err = composerInstall(w, composerJSON); err != nil {
		return
	}

	if err = composerArchive(w, composerJSON, archiveFormat); err != nil {
		return
	}

	sendDownload(w, dir, archiveFormat)
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

	statikFS, err := fs.New()
	if err != nil {
	  appLogger.Error("Failed to start Server", "error", err)
	  return
	}
	router := mux.NewRouter()
	router.Handle("/", http.FileServer(statikFS))
	router.HandleFunc("/vendor/{extension}", VendorHandler).Methods("POST")
	// http.Handle("/", http.StripPrefix("/public/", ))
	http.Handle("/", router)
	appLogger.Info("Starting server", "address", bind, "workingDirectory", uploadsDir)
	if err := http.ListenAndServe(bind, nil); err != nil {
		appLogger.Error("Failed to start Server", "error", err)
	}
}
