// Copyright 2016, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package piazza

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//----------------------------------------------------------

const (
	// ContentTypeJSON is the http content-type for JSON.
	ContentTypeJSON = "application/json"

	// ContentTypeText is the http content-type for plain text.
	ContentTypeText = "text/plain"
)

//----------------------------------------------------------

// Put, because there is no http.Put.
func HTTPPut(url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	client := &http.Client{}
	return client.Do(req)
}

// Delete, because there is no http.Delete.
func HTTPDelete(url string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	return client.Do(req)
}

//---------------------------------------------------------------------

func GinReturnJson(c *gin.Context, resp *JsonResponse) {
	// this just for error checking
	_, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Fatalf("Internal Error: marshalling of %#v", resp)
	}

	c.JSON(resp.StatusCode, resp)

	// If things get worse, try this:
	//    c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	//    c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(raw)))
}

// GetApiKey retrieves the Pz API key for the given server, in this order:
//
// (1) if $PZKEY present, use that
// (2) if ~/.pzkey exists, use that
// (3) error
//
// And no, we don't uspport Windows.
func GetApiKey(pzserver string) (string, error) {

	fileExists := func(s string) bool {
		if _, err := os.Stat(s); os.IsNotExist(err) {
			return false
		}
		return true
	}

	key := os.Getenv("PZKEY")
	if key != "" {
		key = strings.TrimSpace(key)
		return key, nil
	}

	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("Unable read $HOME")
	}

	path := home + "/.pzkey"
	if !fileExists(path) {
		return "", errors.New("Unable to find env var $PZKEY or file $HOME/.pzkey")
	}

	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	data := map[string]string{}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return "", err
	}

	key, ok := data[pzserver]
	if !ok {
		return "", fmt.Errorf("No API key for server %s", pzserver)
	}

	return key, nil
}

// TODO: just for backwards compatability
func GetApiServer() (string, error) {
	server := os.Getenv("PZSERVER")
	if server == "" {
		return "", fmt.Errorf("$PZSERVER not set")
	}
	return server, nil
}

// GetPiazzaServer returns the URL of the $PZSERVER host, i.e. the public entry point.
func GetPiazzaUrl() (string, error) {
	host, err := GetApiServer()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s", DefaultProtocol, host), nil
}

// GetServiceServer returns the host name of the given service, based on $PZSERVER.
func GetPiazzaServiceUrl(serviceName ServiceName) (string, error) {
	pzHost, err := GetApiServer()
	if err != nil {
		return "", err
	}

	i := strings.Index(pzHost, ".")
	if i < 1 {
		return "", fmt.Errorf("Piazza server name is malformed: %s", pzHost)
	}

	serviceHost := string(serviceName) + pzHost[i:]

	return fmt.Sprintf("%s://%s", DefaultProtocol, serviceHost), nil
}

// GetExternalIP returns the "best"(?) IP address we can reasonably get.
// see: http://stackoverflow.com/a/23558495
func GetExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

//---------------------------------------------------------------------

func WaitForService(name ServiceName, url string) error {
	msTime := 0

	for {
		resp, err := http.Get(url)
		defer func() {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()

		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if msTime >= waitTimeoutMs {
			return fmt.Errorf("timed out waiting for service: %s at %s", name, url)
		}

		time.Sleep(waitSleepMs * time.Millisecond)
		msTime += waitSleepMs
	}

	return nil
}

func WaitForServiceToDie(name ServiceName, url string) error {
	msTime := 0

	for {
		resp, err := http.Get(url)
		defer func() {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()

		// we'll accept any error as evidence the service is down
		if err != nil {
			break
		}

		if msTime >= waitTimeoutMs {
			return fmt.Errorf("timed out waiting for service to die: %s at %s", name, url)
		}
		time.Sleep(waitSleepMs * time.Millisecond)
		msTime += waitSleepMs
	}

	return nil
}
