package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const (
	pluginPath = "/plugins/matterbar/notify"
)

var host string
var scheme string
var auth string

// executablePath returns the current path, used to find the path to .json test data files
func executablePath() string {
	executable, err := os.Executable()
	if err != nil {
		fmt.Sprintln(err.Error())
		return ""
	}
	executablePath := filepath.Dir(executable)
	return executablePath
}

// loadJSONFile reads a .json file into bytes
func loadJSONFile(absPath string) []byte {
	bytes, err := ioutil.ReadFile(absPath)
	if err != nil {
		fmt.Println(err.Error())
	}
	return bytes
}

// buildURL creates the absolute URL string
func buildURL() string {
	query := url.Values{}
	query.Set("auth", auth)

	url := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     pluginPath,
		RawQuery: query.Encode(),
	}

	return url.String()
}

// validateFlags ensures that required flags are present and optional flags are valid
func validateFlags() string {
	if auth == "" {
		return "Missing auth token. Usage: `-a <auth-token>`"
	}

	if scheme != "http" && scheme != "https" {
		return fmt.Sprintf(`Invalid scheme "%s". Expected one of {"http", "https"}`, scheme)
	}

	return ""
}

// postData sends an http POST requests for every json file in files
func postData(files []string, url string) {
	for _, f := range files {
		_, filename := filepath.Split(f)
		fmt.Printf("Sending %s to %s...", filename, "localhost")

		requestBody := bytes.NewBuffer(loadJSONFile(f))
		response, err := http.Post(url, "application/json", requestBody)
		if err != nil {
			fmt.Println("http post error")
			return
		}

		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println("body read error")
			return
		}

		if string(responseBody) == "" {
			responseBody = []byte("No response body")
		}

		fmt.Printf("  %s: %s\n", response.Status, responseBody)
	}
}

func init() {
	flag.StringVar(&host, "h", "localhost:8065", "the host to audit. default `localhost:8065`")
	flag.StringVar(&scheme, "s", "http", "the url scheme. default `http`")
	flag.StringVar(&auth, "a", "", "the auth token to use")
}

func main() {
	flag.Parse()
	if errMsg := validateFlags(); errMsg != "" {
		fmt.Println(errMsg)
		return
	}

	executablePath := executablePath()
	if executablePath == "" {
		return
	}

	testdataGlob := filepath.Join(executablePath, "..", "server/testdata/*.json")
	files, err := filepath.Glob(testdataGlob)
	if err != nil {
		return
	}

	url := buildURL()
	postData(files, url)
}
