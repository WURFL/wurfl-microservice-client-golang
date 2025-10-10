/*
Copyright 2019 ScientiaMobile Inc. http://www.scientiamobile.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package wmclient

// Contains all the data structures used by both wm server and client

// JSONInfoData - server and API informations
type JSONInfoData struct {
	WurflAPIVersion  string   `json:"wurfl_api_version"`
	WurflInfo        string   `json:"wurfl_info"`
	WmVersion        string   `json:"wm_version"`
	ImportantHeaders []string `json:"important_headers"`
	StaticCaps       []string `json:"static_caps"`
	VirtualCaps      []string `json:"virtual_caps"`
	Ltime            string   `json:"ltime"`
}

// Request - data object that is sent to the WM server in POST requests
type Request struct {
	LookupHeaders  map[string]string `json:"lookup_headers"`
	RequestedCaps  []string          `json:"requested_caps"`
	RequestedVCaps []string          `json:"requested_vcaps, omitempty"`
	WurflID        string            `json:"wurfl_id, omitempty"`
	TacCode        string            `json:"tac_code, omitempty"`
}

// JSONDeviceData models a WURFL device data in JSON string only format
type JSONDeviceData struct {
	APIVersion      string            `json:"apiVersion"`
	Capabilities    map[string]string `json:"capabilities"`
	Error           string            `json:"error, omitempty"`
	Mtime           int64             `json:"mtime"`           // timestamp of this data structure creation
	Ltime           string            `json:"ltime"`           // time of last wurfl.xml file load
	WmClientVersion string            `json:"wmClientVersion"` // wmclient API version
}

// JSONDeviceDataTyped models a WURFL device data in JSON typed format
type JSONDeviceDataTyped struct {
	APIVersion   string                 `json:"apiVersion"`
	Capabilities map[string]interface{} `json:"capabilities"`
	Error        string                 `json:"error, omitempty"`
	Mtime        int64                  `json:"mtime"`
	Ltime        string                 `json:"ltime"`
}

// JSONMakeModel models simple device "identity" data in JSON format
type JSONMakeModel struct {
	BrandName     string `json:"brand_name"`
	ModelName     string `json:"model_name"`
	MarketingName string `json:"marketing_name,omitempty"`
}

// JSONModelMktName holds model_name and marketing_name
type JSONModelMktName struct {
	ModelName     string `json:"model_name"`
	MarketingName string `json:"marketing_name,omitempty"`
}

// JSONDeviceOsVersions holds device os name and version
type JSONDeviceOsVersions struct {
	OsName    string `json:"device_os"`
	OsVersion string `json:"device_os_version"`
}

// JSONStatsData holds cache statistics and client information
type JSONStatsData struct {
	UserAgentCacheHits   uint64 `json:"user_agent_cache_hits"`
	UserAgentCacheMisses uint64 `json:"user_agent_cache_misses"`
	DeviceIDCacheHits    uint64 `json:"device_id_cache_hits"`
	DeviceIDCacheMisses  uint64 `json:"device_id_cache_misses"`
	WmClientVersion      string `json:"wm_client_version"`
	Timestamp            int64  `json:"timestamp"`
}
