package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/laplaceon/cfbypass"
	log "github.com/sirupsen/logrus"
)

// UserAgent is a customer browser user agent used in every HTTP connections
const UserAgent string = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.62 Safari/537.36"

// Fetch opens a url with a custom client created by user and returns the resulting html page.
// Cannot use the straight http.Get function because need to
// modify headers in order to set a fake user-agent.
func Fetch(url string, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %v", err)
	}

	req.Header.Set("User-Agent", UserAgent)

	log.WithFields(log.Fields{
		"httpMethod":   req.Method,
		"url":          req.URL,
		"httpProtocol": req.Proto,
		"host":         req.Host,
		"headers":      req.Header,
	}).Debug("Successfully built HTTP request")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not launch request: %v", err)
	}

	log.WithFields(log.Fields{
		"httpStatus": resp.Status,
		"headers":    resp.Header,
	}).Debug("Successfully received HTTP response")

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	return resp, nil
}

// BypassCloudflare validates the Clouflare's Javascript challenge.
// Once the challenge is resolved, it passes 2 new Cloudflare cookies to the client
// so every new requests won't be challenged anymore.
func BypassCloudflare(url url.URL, client *http.Client) (*http.Client, error) {
	cookies := client.Jar.Cookies(&url)
	cookies = append(cookies, cfbypass.GetTokens(url.String(), UserAgent, "")...)
	client.Jar.SetCookies(&url, cookies)
	log.Debug("Successfully added Clouflare cookies to client")

	return client, nil
}

// DlFile downloads the torrent with a custom client created by user and returns the path of
// downloaded file.
// The name of the downloaded file is made up of the search arguments + the
// Unix timestamp to avoid collision. Ex: comte_de_montecristo_1581064034469619222.torrent
func DlFile(fileURL string, in string, client *http.Client) (string, error) {
	// Get torrent file name from url
	fileName := strings.Replace(in, " ", "_", -1)
	fileName += "_" + strconv.Itoa(int(time.Now().UnixNano())) + ".torrent"

	// Create local torrent file
	out, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("could not create the torrent file named %s: %v", fileName, err)
	}
	defer out.Close()

	// Download torrent
	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("could not create request: %v", err)
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not download the torrent file: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return "", fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	// Save torrent to disk
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not save the torrent file to disk: %v", err)
	}

	// Get absolute file path of torrent
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", fmt.Errorf("could not retrieve current directory of saved filed: %v", err)
	}
	filePath := dir + "/" + fileName

	return filePath, nil
}
