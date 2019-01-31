package main

/**
 * Client for Compozipd
 */
import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

var (
	serverHost    string
	serverPort    string
	filename      string
	archiveFormat string
	client        *http.Client
)

func init() {
	flag.StringVar(&serverHost, "host", "localhost", "compozipd Server address")
	flag.StringVar(&serverPort, "port", "80", "compozipd Server address")
	flag.StringVar(&filename, "c", "composer.json", "Composer.json file to upload")
	flag.StringVar(&archiveFormat, "f", "zip", "Archive format. One of 'zip', 'tar'")
}

func main() {
	flag.Parse()

	client = &http.Client{}
	uploadURL := fmt.Sprintf("http://%s:%s/vendor/%s",
		serverHost,
		serverPort,
		archiveFormat,
	)
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	// this step is very important
	fileWriter, err := bodyWriter.CreateFormFile("composer", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return
	}

	fh, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Failed to open file. Got error: %s \n", err)
		return
	}
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		fmt.Printf("Failed to upload file. Got error: %s \n", err)
		return
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	fmt.Printf("Uploading %s ...\n", filename)
	response, err := client.Post(uploadURL, contentType, bodyBuf)
	if err != nil {
		fmt.Printf("Failed to upload file. Got error: %s \n", err)
		return
	}
	defer response.Body.Close()
	fmt.Printf("Downloading vendor archive (vendor.%s)...\n", archiveFormat)
	if response.StatusCode == 400 || response.StatusCode == 500 {
		fmt.Fprintf(os.Stderr, "Failed to download vendor.%s. Got error: %s\n",
			archiveFormat,
			response.Body,
		)
		return
	}
	vendorZIPBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to extract vendor.%s. Got error: %s\n",
			archiveFormat,
			response.Body,
		)
		return
	}

	baseDirectory, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	vendorArchiveName := fmt.Sprintf("vendor.%s", archiveFormat)
	vendorFullPath := path.Join(baseDirectory, vendorArchiveName)
	err = ioutil.WriteFile(vendorFullPath, vendorZIPBytes, 0664)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write %s. to %s Got error: %s\n",
			vendorArchiveName,
			vendorFullPath,
			response.Body,
		)
		return
	}
	fmt.Printf("Downloaded vendor archive to %s", vendorFullPath)
	// TODO: Extract? fmt.Println("Extracting vendor archive to ./vendor ...")
}
