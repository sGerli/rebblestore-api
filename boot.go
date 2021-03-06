package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
)

const (
	LOCAL_BOOT_URI  string = "http://127.0.0.1:8080"
	PEBBLE_BOOT_URL string = "https://boot.getpebble.com/api/config/"
	STORE_URI       string = "https://store.rebble.io"
)

// BootJSON is just a Go container object for the JSON response.
type BootJSON struct {
	Config BootConfig `json:"config"`
}

// BootConfig contains the webviews from the JSON file.
type BootConfig struct {
	Algolia        json.RawMessage   `json:"algolia"`
	AppMeta        json.RawMessage   `json:"app_meta"`
	Authentication json.RawMessage   `json:"authentication"`
	Cohorts        json.RawMessage   `json:"cohorts"`
	Developer      json.RawMessage   `json:"developer"`
	Health         json.RawMessage   `json:"health"`
	Href           string            `json:"href"`
	Id             string            `json:"id"`
	KeenIo         json.RawMessage   `json:"keen_io"`
	LinkedServices json.RawMessage   `json:"linked_services"`
	Links          json.RawMessage   `json:"links"`
	Locker         json.RawMessage   `json:"locker"`
	Notifications  json.RawMessage   `json:"notifications"`
	SupportRequest json.RawMessage   `json:"support_request"`
	Timeline       json.RawMessage   `json:"timeline"`
	TreasureData   json.RawMessage   `json:"treasure_data"`
	Voice          json.RawMessage   `json:"voice"`
	Webviews       map[string]string `json:"webviews"`
}

// WebviewConfig contains the webviews in-which we would like to override.
type WebviewsConfig struct {
	FAQ                  string `json:"support/faq"`
	Application          string `json:"appstore/application"`
	ApplicationChangelog string `json:"appstore/application_changelog"`
	DeveloperApps        string `json:"appstore/developer_apps"`
	Watchfaces           string `json:"appstore/watchfaces"`
	Watchapps            string `json:"appstore/watchapps"`
}

// BootHandler is based off of [@afourney|https://github.com/afourney]'s
// development bootstrap override.
func BootHandler(w http.ResponseWriter, r *http.Request) {
	WriteCommonHeaders(w)
	// Get a store uri from the request and determine if it matches a valid URI
	store_uri := r.URL.Query().Get("store_uri")
	if _, err := url.Parse(store_uri); err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Invalid store_uri query"))
		return
	}

	// Copying the URL Query to modify it later
	urlquery := r.URL.Query()

	// If the user didn't specify a store_uri, use the pebble server
	if store_uri == "" {
		store_uri = STORE_URI
	} else {
		urlquery.Del("store_uri")
	}

	var request_url string

	// Build up the request URL
	os := mux.Vars(r)["os"]
	if os == "android" || os == "ios" {
		request_url = fmt.Sprintf("%s%s/%s?%s", PEBBLE_BOOT_URL, os, mux.Vars(r)["path"], urlquery.Encode())
	} else {
		w.Write([]byte("Invalid OS parameter"))
		return
	}
	// Make a request to an external server then parse the request
	req, err := http.Get(request_url)
	if err != nil {
		log.Fatal("Could not contact api:", err)
	}
	if req.StatusCode < 200 || req.StatusCode > 299 {
		log.Println("API Answered with status code", req.StatusCode, "- carrying on anyway...")
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Fatal("Could not read api response:", err)
	}

	// Decode the JSON data
	response := &BootJSON{}
	err = json.Unmarshal(data, response)
	if err != nil {
		log.Println("Could not parse api response: ", err)
		w.Write(data)
		return
	}

	// Replace items in the JSON object, then prepare to output it
	response.Config.Webviews["support/faq"] = fmt.Sprintf("%s/faq", store_uri)
	response.Config.Webviews["appstore/application"] = fmt.Sprintf("%s/application/$$id$$?pebble_color=$$pebble_color$$&hardware=$$hardware$$&uid=$$user_id$$&mid=$$phone_id$$&pid=$$pebble_id$$&$$extras$$", store_uri)
	response.Config.Webviews["appstore/application_changelog"] = fmt.Sprintf("%s/changelog/$$id$$?pebble_color=$$pebble_color$$&hardware=$$hardware$$&uid=$$user_id$$&mid=$$phone_id$$&pid=$$pebble_id$$&$$extras$$", store_uri)
	response.Config.Webviews["appstore/developer_apps"] = fmt.Sprintf("%s/developer/$$id$$?pebble_color=$$pebble_color$$&hardware=$$hardware$$&uid=$$user_id$$&mid=$$phone_id$$&pid=$$pebble_id$$&$$extras$$", store_uri)
	response.Config.Webviews["appstore/watchfaces"] = fmt.Sprintf("%s/watchfaces?pebble_color=$$pebble_color$$&hardware=$$hardware$$&uid=$$user_id$$&mid=$$phone_id$$&pid=$$pebble_id$$&$$extras$$", store_uri)
	response.Config.Webviews["appstore/watchapps"] = fmt.Sprintf("%s/watchapps?pebble_color=$$pebble_color$$&hardware=$$hardware$$&uid=$$user_id$$&mid=$$phone_id$$&pid=$$pebble_id$$&$$extras$$", store_uri)
	response.Config.Href = LOCAL_BOOT_URI + r.URL.Path
	response.Config.Id = strings.Replace(r.URL.Path, "/boot/", "", -1)

	data, err = json.MarshalIndent(response, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	// Send the JSON object back to the user
	w.Header().Add("content-type", "application/json")
	w.Write(data)
}
