package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path"
	"strings"

	"github.com/gorilla/mux"
)

func parseURLParameters(w http.ResponseWriter, r *http.Request) (string, error) {
	params := mux.Vars(r)
	archiveFormat := strings.ToLower(params["extension"])
	if strings.Compare(archiveFormat, "zip") != 0 && strings.Compare(archiveFormat, "tar") != 0 {
		appLogger.Warn("Invalid format type specified", "extension", archiveFormat)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprint(w, "Please specify valid archive type, either 'zip' or 'tar'")
		return "", errors.New("Invalid file type specified")
	}
	return archiveFormat, nil
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
	multiPartForm, err := multipartReader.ReadForm(MaxMemoryBytes)
	if err != nil {
		appLogger.Error("Failed to parse Multipart-Form.", "error", err)
		w.WriteHeader(400)
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprintf(w, "File could not be read in request")
		return nil, nil, err
	}

	composerFiles, ok := multiPartForm.File["composer"]
	if !ok {
		appLogger.Error("Multipart form-data didn't contain 'composer' file.")
		return nil, nil, errors.New("Please provide 'composer' file in the form-data")
	}

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
