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

package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/wurfl/wurfl-microservice-client-golang/v2/scientiamobile/wmclient"
)

// Implements sort.Interface for []wmclient.JSONModelMktName
type ByModelName []wmclient.JSONModelMktName

func (c ByModelName) Len() int {
	return len(c)
}

func (c ByModelName) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c ByModelName) Less(i, j int) bool {
	return c[i].ModelName < c[j].ModelName
}

func main() {
	var err error

	// First we need to create a WM client instance, to connect to our WM server API at the specified host and port.
	ClientConn, err := wmclient.Create("http", "localhost", "80", "")
	if err != nil {
		// problems such as network errors  or internal server problems
		log.Fatal("wmclient.Create returned :", err.Error())
	}
	// By setting the cache size we are also activating the caching option in WM client. In order to not use cache, you just to need to omit setCacheSize call
	ClientConn.SetCacheSize(100000)

	// We ask Wm server API for some Wm server info such as server API version and info about WURFL API and file used by WM server.
	info, ierr := ClientConn.GetInfo()
	if ierr != nil {
		fmt.Println("Error getting server info: " + ierr.Error())
	} else {
		fmt.Println("WURFL Microservice information:")
		fmt.Println("Server version: " + info.WmVersion)
		fmt.Println("WURFL API version: " + info.WurflAPIVersion)
		fmt.Printf("WURFL file info: %s \n", info.WurflInfo)
	}

	// set the capabilities we want to receive from WM server
	// Static capabilities
	sCapsList := strings.Fields("model_name brand_name")
	// Virtual capabilities
	vCapsList := strings.Fields("is_smartphone form_factor")
	ClientConn.SetRequestedStaticCapabilities(sCapsList)
	ClientConn.SetRequestedVirtualCapabilities(vCapsList)

	// ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 5_0 like Mac OS X) AppleWebKit/534.46 (KHTML, like Gecko) Version/5.1 Mobile/9A334 Safari/7534.48.3"
	// Perform a device detection calling WM server API
	// JSONDeviceData, callerr := ClientConn.LookupUserAgent(ua)

	// or use the a http request struct to perform the detection.
	request, err := http.NewRequest("GET", "www.gitub.com", nil)
	if err == nil {
		request.Header.Add("Content-Type", "application/json")
		request.Header.Add("Accept", "text/html, application/xml;q=0.9, application/xhtml+xml, image/png, image/webp, image/jpeg, image/gif, image/x-xbitmap, */*;q=0.1")
		request.Header.Add("Accept-Encoding", "gzip, deflate")
		request.Header.Add("Accept-Language", "en")
		request.Header.Add("Device-Stock-Ua", "Mozilla/5.0 (Linux; Android 8.1.0; SM-J610G Build/M1AJQ; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/69.0.3497.100 Mobile Safari/537.36")
		request.Header.Add("Forwarded", "for=\"110.54.224.195:36350\"")
		request.Header.Add("Referer", "https://www.cram.com/flashcards/labor-and-delivery-questions-889210")
		request.Header.Add("User-Agent", "Opera/9.80 (Android; Opera Mini/51.0.2254/184.121; U; en) Presto/2.12.423 Version/12.16")
		request.Header.Add("X-Clacks-Overhead", "GNU ph")
		request.Header.Add("X-Forwarded-For", "110.54.224.195, 82.145.210.235")
		request.Header.Add("X-Operamini-Features", "advanced, camera, download, file_system, folding, httpping, pingback, routing, touch, viewport")
		request.Header.Add("X-Operamini-Phone", "Android #")
		request.Header.Add("X-Operamini-Phone-Ua", "Mozilla/5.0 (Linux; Android 8.1.0; SM-J610G Build/M1AJQ; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/69.0.3497.100 Mobile Safari/537.36")
	}

	JSONDeviceData, callerr := ClientConn.LookupRequest(*request)

	if callerr != nil {
		// Applicative error, ie: invalid input provided
		log.Fatalf("Error getting device data %s\n", callerr.Error())
	}

	// Let's get the device capabilities and print some of them
	fmt.Printf("WURFL device id : %s\n", JSONDeviceData.Capabilities["wurfl_id"])

	// print brand & model (static capabilities)
	fmt.Printf("This device is a : %s %s\n", JSONDeviceData.Capabilities["brand_name"], JSONDeviceData.Capabilities["model_name"])

	// check if device is a smartphone (a virtual capability)
	if JSONDeviceData.Capabilities["is_smartphone"] == "true" {
		fmt.Println("This is a smartphone")
	}
	fmt.Printf("This device form_factor is %s\n", JSONDeviceData.Capabilities["form_factor"])

	// Get all the device manufacturers, and print the first twenty
	deviceMakes, err := ClientConn.GetAllDeviceMakes()
	if err != nil {
		log.Fatalf("Error getting device data %s\n", err.Error())
	}

	var limit = 20
	fmt.Printf("Print the first %d Brand of %d\n", limit, len(deviceMakes))

	// Sort the device manufacturer names
	sort.Strings(deviceMakes)

	for _, deviceMake := range deviceMakes[0:limit] {
		fmt.Printf(" - %s\n", deviceMake)
	}

	// Now call the WM server to get all device model and marketing names produced by Apple
	fmt.Println("Print all Model for the Apple Brand")
	modelMktNames, err := ClientConn.GetAllDevicesForMake("Apple")

	if err != nil {
		log.Fatalf("Error getting device data %s\n", err.Error())
	}

	// Sort modelMktNames objects by their model name
	sort.Sort(ByModelName(modelMktNames))

	for _, modelMktName := range modelMktNames {
		fmt.Printf(" - %s %s\n", modelMktName.ModelName, modelMktName.MarketingName)
	}

	// Now call the WM server to get all operative system names
	fmt.Println("Print the list of OSes")
	oses, err := ClientConn.GetAllOSes()

	if err != nil {
		log.Fatalf("Error getting device data %s\n", err.Error())
	}

	// Sort and print all OS names
	sort.Strings(oses)

	for _, os := range oses {
		fmt.Printf(" - %s\n", os)
	}

	// Let's call the WM server to get all version of the Android OS
	fmt.Println("Print all versions for the Android OS")
	versions, err := ClientConn.GetAllVersionsForOS("Android")

	if err != nil {
		log.Fatalf("Error getting device data %s\n", err.Error())
	}

	// Sort all Android version numbers and print them.
	sort.Strings(versions)

	for _, version := range versions {
		fmt.Printf(" - %s\n", version)
	}
}
