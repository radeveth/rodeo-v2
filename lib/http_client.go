package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// ErrUnauthorized is returned by GetJSON when it gets a status code of 401
var ErrUnauthorized = errors.New("HTTP Client: Unauthorized request")

// GetJSON fetches a given url with provided headers and parses the answer as JSON to the response object
func GetJSON(url string, response interface{}, headers map[string]string) {
	Check(GetJSONErr(url, response, headers))
}

// GetJSONErr fetches a given url with provided headers and parses the answer as JSON to the response object
func GetJSONErr(url string, response interface{}, headers map[string]string) error {
	return getJSON("GET", url, "", response, headers, 0)
}

func PostForm(url string, response interface{}, headers map[string]string, body map[string]string) {
	Check(PostFormErr(url, response, headers, body))
}

func PostFormErr(_url string, response interface{}, headers map[string]string, body map[string]string) error {
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	b := url.Values{}
	for k, v := range body {
		b.Set(k, v)
	}
	return getJSON("POST", _url, b.Encode(), response, headers, 0)
}

func PostJSON(url string, response interface{}, headers map[string]string, body interface{}) {
	Check(PostJSONErr(url, response, headers, body))
}

func PostJSONErr(url string, response interface{}, headers map[string]string, body interface{}) error {
	headers["Content-Type"] = "application/json"
	bs, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return getJSON("POST", url, string(bs), response, headers, 0)
}

func getJSON(method, url, bodyString string, response interface{}, headers map[string]string, try int) error {
	retry := func(err error) error {
		if try >= 3 {
			return fmt.Errorf("fetch: out of retries: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
		return getJSON(method, url, bodyString, response, headers, try+1)
	}
	req, err := http.NewRequest(method, url, bytes.NewBufferString(bodyString))
	if err != nil {
		return fmt.Errorf(`fetching "%s": %v`, url, err)
	}
	req.Header.Set("Accept", "application/json")
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return retry(fmt.Errorf(`fetching "%s": %v`, url, err))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return retry(fmt.Errorf(`fetching "%s": got status code %v (%s)`, url, resp.StatusCode, string(body)))
	}
	errorStart := `{"error":{`
	if len(body) > len(errorStart) && string(body)[0:len(errorStart)] == errorStart {
		return fmt.Errorf(`fetching "%s": got error %v`,
			url, body)
	}
	return json.Unmarshal(body, response)
}
