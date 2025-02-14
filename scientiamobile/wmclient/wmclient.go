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

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
)

// userAgentHeader is the User-Agent header name
const userAgentHeader = "User-Agent"
const deviceDefaultCacheSize = 20000

// default timeouts
const defaultConnTimeout = time.Duration(10 * time.Second)
const defaultTransferTimeout = time.Duration(60 * time.Second)

// WmClient holds http connection data to  WM server and the list of static and virtual capabilities it must return in response.
type WmClient struct {
	scheme      string
	host        string
	port        string
	baseURI     string
	StaticCaps  []string
	VirtualCaps []string
	// requested*Caps are used in the lookup requests, accessible via the SetRequested[...] methods
	requestedStaticCaps  []string
	requestedVirtualCaps []string
	httpClient           *http.Client
	ImportantHeaders     []string
	deviceCache          *lru.Cache
	userAgentCache       *lru.Cache
	lruDeviceCS          sync.Mutex
	lruUserAgentCS       sync.Mutex
	connTimeout          time.Duration
	transferTimeout      time.Duration
	mkMdMutex            sync.Mutex // protects the data shared data structure below
	mkModels             []JSONMakeModel
	deviceMakesMutex     sync.Mutex // protects the data shared data structure below
	deviceMakes          []string
	deviceMakesMap       map[string][]JSONModelMktName

	deviceOsesMutex sync.Mutex // protects the data shared data structure below
	deviceOses      []string
	deviceOsVerMap  map[string][]string

	clientLtime string
}

// GetAPIVersion returns the version number of WM Client API
func GetAPIVersion() string {
	return "2.1.3"
}

// creates a new http.Client with the specified timeouts
func createHTTPClient(connTimeout time.Duration, transferTimeout time.Duration) *http.Client {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: connTimeout,
		}).Dial,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	cl := &http.Client{
		Timeout:   transferTimeout,
		Transport: netTransport,
	}

	return cl
}

// Create : creates object, checks for server visibility
func Create(Scheme string, Host string, Port string, BaseURI string) (*WmClient, error) {
	client := &WmClient{}
	if len(Scheme) > 0 {
		client.scheme = Scheme
	} else {
		client.scheme = "http"
	}

	client.host = Host
	client.port = Port

	client.baseURI = BaseURI
	client.httpClient = createHTTPClient(defaultConnTimeout, defaultTransferTimeout)

	// Test server connection and save important headers taken using getInfo function
	data, err := client.GetInfo()
	if err != nil {
		return nil, err
	}

	client.ImportantHeaders = data.ImportantHeaders
	client.StaticCaps = data.StaticCaps
	client.VirtualCaps = data.VirtualCaps
	sort.Strings(client.StaticCaps)
	sort.Strings(client.VirtualCaps)
	return client, nil
}

// SetRequestedStaticCapabilities - set list of standard static capabilities to return
func (c *WmClient) SetRequestedStaticCapabilities(CapsList []string) {

	if CapsList == nil {
		c.requestedStaticCaps = nil
		c.clearCache()
		return
	}

	var capNames = make([]string, 0, 16)
	for _, name := range CapsList {

		if c.HasStaticCapability(name) {
			capNames = append(capNames, name)
		}
	}

	if capNames != nil && len(capNames) > 0 {
		c.requestedStaticCaps = capNames
		c.clearCache()
	}
}

// SetRequestedVirtualCapabilities - set list of virtual capabilities to return
func (c *WmClient) SetRequestedVirtualCapabilities(CapsList []string) {
	if CapsList == nil {
		c.requestedVirtualCaps = nil
		c.clearCache()
		return
	}

	var vcapNames = make([]string, 0, 4)
	for _, name := range CapsList {
		if c.HasVirtualCapability(name) {
			vcapNames = append(vcapNames, name)
		}
	}

	if vcapNames != nil && len(vcapNames) > 0 {
		c.requestedVirtualCaps = vcapNames
		c.clearCache()
	}
}

// SetRequestedCapabilities - set the given capability names to the set they belong
func (c *WmClient) SetRequestedCapabilities(CapsList []string) {
	if CapsList == nil {
		c.requestedVirtualCaps = nil
		c.requestedStaticCaps = nil
		c.clearCache()
		return
	}

	capNames := make([]string, 0, 16)
	vcapNames := make([]string, 0, 4)
	for _, name := range CapsList {
		if c.HasStaticCapability(name) {
			capNames = append(capNames, name)
		} else if c.HasVirtualCapability(name) {
			vcapNames = append(vcapNames, name)
		}
	}
	c.requestedStaticCaps = capNames
	c.requestedVirtualCaps = vcapNames
	c.clearCache()
}

// SetCacheSize : set UA cache size
func (c *WmClient) SetCacheSize(uaMaxEntries int) {
	c.userAgentCache = lru.New(uaMaxEntries)
	c.deviceCache = lru.New(deviceDefaultCacheSize)
}

// clearCache Removes all entries from WM client cache, every cache is cleared using its own mutex, to avoid goroutines to use it while we are clearing it
func (c *WmClient) clearCache() {

	c.lruUserAgentCS.Lock()
	if c.userAgentCache != nil && c.userAgentCache.Len() > 0 {
		c.userAgentCache.Clear()
	}
	c.lruUserAgentCS.Unlock()

	c.lruDeviceCS.Lock()
	if c.deviceCache != nil && c.deviceCache.Len() > 0 {
		c.deviceCache.Clear()
	}
	c.lruDeviceCS.Unlock()

	c.mkMdMutex.Lock()
	c.mkModels = nil
	c.mkMdMutex.Unlock()

	c.deviceMakesMutex.Lock()
	c.deviceMakes = nil
	c.deviceMakesMap = nil
	c.deviceMakesMutex.Unlock()

	c.deviceOsesMutex.Lock()
	c.deviceOses = nil
	c.deviceOsVerMap = nil
	c.deviceOsesMutex.Unlock()
}

// GetActualCacheSizes return the values of cache size. The first value being the device-id based cache, the second value being
// the size of the headers-based one
func (c *WmClient) GetActualCacheSizes() (int, int) {
	var dSize int
	var uaSize int

	// Lock the caches with their own mutex, so that other goroutines cannot clear it while another is reading its size
	c.lruDeviceCS.Lock()
	if c.deviceCache != nil {
		dSize = c.deviceCache.Len()
	}
	c.lruDeviceCS.Unlock()

	c.lruUserAgentCS.Lock()
	if c.userAgentCache != nil {
		uaSize = c.userAgentCache.Len()
	}
	c.lruUserAgentCS.Unlock()

	return dSize, uaSize
}

// HasStaticCapability - returns true if the given CapName exist in this client' static capability set, false otherwise
func (c *WmClient) HasStaticCapability(CapName string) bool {
	return sliceHasValue(c.StaticCaps, CapName)
}

// HasVirtualCapability - returns true if the given CapName exist in this client' virtual capability set, false otherwise
func (c *WmClient) HasVirtualCapability(CapName string) bool {
	return sliceHasValue(c.VirtualCaps, CapName)
}

// checks whether the given value is present in the given slice of strings
func sliceHasValue(slist []string, value string) bool {
	if slist == nil {
		return false
	}

	index := sort.SearchStrings(slist, value)
	// When this function DOES NOT find the searched value inside the slice, it returns the index where it SHOULD be in case it was found.
	// Thus, a check of equality between the expected value and the actual one is needed
	return index < len(slist) && value == slist[index]
}

// LookupRequest - detects a device and returns its data in JSON format
func (c *WmClient) LookupRequest(request http.Request) (*JSONDeviceData, error) {

	jrequest := Request{LookupHeaders: make(map[string]string)}

	// copy headers
	for i := 0; i < len(c.ImportantHeaders); i++ {
		name := c.ImportantHeaders[i]
		h := request.Header.Get(name)
		if h != "" {
			jrequest.LookupHeaders[name] = h
		}
	}

	// Do a cache lookup
	if c.userAgentCache != nil {

		c.lruUserAgentCS.Lock()
		value, ok := c.userAgentCache.Get(c.getUserAgentCacheKey(jrequest.LookupHeaders))
		c.lruUserAgentCS.Unlock()

		if ok {
			jdd := value.(*JSONDeviceData)
			return jdd, nil
		}
	}

	jrequest.RequestedCaps = c.requestedStaticCaps
	jrequest.RequestedVCaps = c.requestedVirtualCaps

	deviceData, err := c.internalLookup(jrequest, "/v2/lookuprequest/json")

	if err == nil {
		// check if server WURFL.xml has been updated and, if so, clear caches
		c.clearCachesIfNeeded(deviceData.Ltime)

		// lock and add element
		if c.userAgentCache != nil {
			c.lruUserAgentCS.Lock()
			c.userAgentCache.Add(c.getUserAgentCacheKey(jrequest.LookupHeaders), deviceData)
			c.lruUserAgentCS.Unlock()
		}
	}

	return deviceData, err
}

// LookupHeaders - detects a device and returns its data in JSON format
func (c *WmClient) LookupHeaders(headers map[string]string) (*JSONDeviceData, error) {

	jrequest := Request{LookupHeaders: make(map[string]string)}

	// first: make all headers lowercase
	var lowerKeyMap = make(map[string]string)
	for k, v := range headers {
		lowerKeyMap[strings.ToLower(k)] = v
	}

	// copy headers
	for i := 0; i < len(c.ImportantHeaders); i++ {
		name := c.ImportantHeaders[i]
		h := lowerKeyMap[strings.ToLower(name)]
		if h != "" {
			jrequest.LookupHeaders[name] = h
		}
	}

	// Do a cache lookup
	if c.userAgentCache != nil {

		c.lruUserAgentCS.Lock()
		value, ok := c.userAgentCache.Get(c.getUserAgentCacheKey(jrequest.LookupHeaders))
		c.lruUserAgentCS.Unlock()

		if ok {
			jdd := value.(*JSONDeviceData)
			return jdd, nil
		}
	}

	jrequest.RequestedCaps = c.requestedStaticCaps
	jrequest.RequestedVCaps = c.requestedVirtualCaps

	deviceData, err := c.internalLookup(jrequest, "/v2/lookuprequest/json")

	if err == nil {
		// check if server WURFL.xml has been updated and, if so, clear caches
		c.clearCachesIfNeeded(deviceData.Ltime)

		// lock and add element
		if c.userAgentCache != nil {
			c.lruUserAgentCS.Lock()
			c.userAgentCache.Add(c.getUserAgentCacheKey(jrequest.LookupHeaders), deviceData)
			c.lruUserAgentCS.Unlock()
		}
	}

	return deviceData, err
}

// LookupUserAgent - Searches WURFL device data using the given user-agent for detection
func (c *WmClient) LookupUserAgent(userAgent string) (*JSONDeviceData, error) {

	// First: cache lookup
	headers := map[string]string{userAgentHeader: userAgent}

	if c.userAgentCache != nil {

		c.lruUserAgentCS.Lock()
		value, ok := c.userAgentCache.Get(c.getUserAgentCacheKey(headers))
		c.lruUserAgentCS.Unlock()

		if ok {
			jdd := value.(*JSONDeviceData)
			return jdd, nil
		}
	}

	var jsonRequest = Request{LookupHeaders: make(map[string]string)}

	// Add user-agent to the Request object
	jsonRequest.LookupHeaders[userAgentHeader] = userAgent
	jsonRequest.RequestedCaps = c.requestedStaticCaps
	jsonRequest.RequestedVCaps = c.requestedVirtualCaps

	deviceData, err := c.internalLookup(jsonRequest, "/v2/lookupuseragent/json")
	if err == nil {
		// check if server WURFL.xml has been updated and, if so, clear caches
		c.clearCachesIfNeeded(deviceData.Ltime)

		// we need to lock when writing since cache is not thread safe
		if c.userAgentCache != nil {
			c.lruUserAgentCS.Lock()
			c.userAgentCache.Add(c.getUserAgentCacheKey(headers), deviceData)
			c.lruUserAgentCS.Unlock()
		}
	}

	return deviceData, err
}

// LookupDeviceID - Searches WURFL device data using its wurfl_id value
func (c *WmClient) LookupDeviceID(deviceID string) (*JSONDeviceData, error) {

	// First: cache lookup
	if c.deviceCache != nil {
		c.lruDeviceCS.Lock()
		value, ok := c.deviceCache.Get(deviceID)
		c.lruDeviceCS.Unlock()

		if ok {
			jdd := value.(*JSONDeviceData)
			return jdd, nil
		}
	}

	var jsonRequest = Request{}
	jsonRequest.WurflID = deviceID
	jsonRequest.RequestedCaps = c.requestedStaticCaps
	jsonRequest.RequestedVCaps = c.requestedVirtualCaps

	deviceData, err := c.internalLookup(jsonRequest, "/v2/lookupdeviceid/json")
	if err == nil {

		// check if server WURFL.xml has been updated and, if so, clear caches
		c.clearCachesIfNeeded(deviceData.Ltime)

		if c.deviceCache != nil {
			// we need to lock when writing since cache is not thread safe
			c.lruDeviceCS.Lock()
			c.deviceCache.Add(deviceID, deviceData)
			c.lruDeviceCS.Unlock()
		}
	}

	return deviceData, err

}

// GetInfo - Returns information about the running WM server and API
func (c *WmClient) GetInfo() (*JSONInfoData, error) {
	var info = JSONInfoData{}

	var body, berr = c.internalGet("/v2/getinfo/json")
	if berr != nil {
		return nil, berr
	}

	var merror = json.Unmarshal(body, &info)
	if merror != nil {

		return nil, merror
	}

	if !checkData(&info) {
		return nil, errors.New("server returned empty data or a wrong json format")
	}

	// check if server WURFL.xml has been updated and, if so, clear caches
	c.clearCachesIfNeeded(info.Ltime)

	return &info, nil
}

// DestroyConnection - Disposes resources used in connection to server and clears cache and other shared data structures
func (c *WmClient) DestroyConnection() {
	if c != nil {

		c.clearCache()
		c.mkModels = nil
		c.httpClient = nil
		c = nil
	}
}

func (c *WmClient) createURL(path string) string {

	url := c.scheme + "://" + c.host
	if len(c.port) > 0 {
		url += ":" + c.port
	}

	if len(c.baseURI) > 0 {
		return url + "/" + c.baseURI + path
	}
	return url + path
}

// Performs a GET request and returns the response body as a byte array JSON that can be unmarshalled
func (c *WmClient) internalGet(endpoint string) ([]byte, error) {
	url := c.createURL(endpoint)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, reserr := c.httpClient.Do(request)
	if reserr != nil {
		return nil, reserr
	}

	defer res.Body.Close()

	var body, berr = ioutil.ReadAll(res.Body)
	if berr != nil {
		return nil, berr
	}

	return body, nil
}

func (c *WmClient) internalLookup(request Request, path string) (*JSONDeviceData, error) {
	var deviceData = JSONDeviceData{}
	url := c.createURL(path)

	reqbody, merr := json.Marshal(request)
	if merr != nil {
		return nil, merr
	}

	httpreq, herr := http.NewRequest("POST", url, bytes.NewBuffer(reqbody))
	if herr != nil {
		return nil, herr
	}

	httpreq.Header.Set("User-Agent", getWmClientUserAgent(httpreq.UserAgent()))

	res, err := c.httpClient.Do(httpreq)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var resbody, berr = ioutil.ReadAll(res.Body)
	if berr != nil {
		return nil, berr
	}

	var umerr = json.Unmarshal(resbody, &deviceData)
	if umerr != nil {
		return nil, umerr
	}

	// check for error messages in json and return it with data from device
	if len(deviceData.Error) > 0 {
		errMsg := deviceData.Error
		deviceData.Error = ""
		return &deviceData, errors.New("Received error from WM server: " + errMsg)
	}

	return &deviceData, nil
}

func getWmClientUserAgent(userAgent string) string {
	return userAgent + "go-wmclient-api-" + GetAPIVersion()
}

func (c *WmClient) getUserAgentCacheKey(headers map[string]string) string {
	key := ""
	// Using important headers array preserves header name order
	for _, hname := range c.ImportantHeaders {
		key += headers[hname]
	}
	md5Sum := md5.Sum([]byte(key))
	return hex.EncodeToString(md5Sum[:])
}

func checkData(data *JSONInfoData) bool {
	// These check ensure received data is OK
	return len(data.WmVersion) > 0 && len(data.WurflAPIVersion) > 0 && len(data.WurflInfo) > 0 &&
		(len(data.StaticCaps) > 0 || len(data.VirtualCaps) > 0)
}

// SetHTTPTimeout sets the connection and transfer timeouts for this client in seconds.
// This function should be called before performing any connection to WM server
func (c *WmClient) SetHTTPTimeout(connection int, transfer int) {
	if connection <= 0 {
		c.connTimeout = defaultConnTimeout
	} else {
		c.connTimeout = time.Duration(time.Duration(connection) * time.Second)
	}

	if transfer <= 0 {
		c.transferTimeout = defaultTransferTimeout
	} else {
		c.transferTimeout = time.Duration(time.Duration(transfer) * time.Second)
	}

	c.httpClient = createHTTPClient(c.connTimeout, c.transferTimeout)

}

// GetAllOSes returns a slice of all devices device_os capabilities in WM server
func (c *WmClient) GetAllOSes() ([]string, error) {

	err := c.loadDeviceOsesData()

	if err != nil && len(c.deviceOses) > 0 {
		return nil, err
	}

	c.deviceOsesMutex.Lock()
	retVal := c.deviceOses // it's not possible to return into mutex
	c.deviceOsesMutex.Unlock()
	return retVal, nil
}

// GetAllVersionsForOS returns a slice of an aggregate containing device_os_version for the given os_name
func (c *WmClient) GetAllVersionsForOS(osName string) ([]string, error) {

	err := c.loadDeviceOsesData()

	if err != nil && len(c.deviceOses) > 0 {
		return nil, err
	}

	c.deviceOsesMutex.Lock()
	if val, ok := c.deviceOsVerMap[osName]; ok {
		c.deviceOsesMutex.Unlock()
		// Now, remove all empty version fields
		osval := make([]string, 0)
		for _, v := range val {
			if v != "" {
				osval = append(osval, v)
			}
		}
		return osval, nil
	}
	c.deviceOsesMutex.Unlock() // unlock here is if block is not traversed

	return nil, errors.New(fmt.Sprintf("Error getting data from WM server: %s does not exist", osName))
}

func (c *WmClient) loadDeviceOsesData() error {
	// We lock the shared makeModel cache
	c.deviceOsesMutex.Lock()
	if c.deviceOses != nil && len(c.deviceOses) > 0 {
		defer c.deviceOsesMutex.Unlock()
		return nil
	}

	// if makeModel cache is empty unlock it
	c.deviceOsesMutex.Unlock()

	osVersionModels := make([]JSONDeviceOsVersions, 1000)
	var body, berr = c.internalGet("/v2/alldeviceosversions/json")
	if berr != nil {
		return berr
	}

	var merror = json.Unmarshal(body, &osVersionModels)
	if merror != nil {
		return merror
	}
	var ovMap = make(map[string][]string, 0)
	var ov = make([]string, 0)

	for _, ovModel := range osVersionModels {

		if _, ok := ovMap[ovModel.OsName]; !ok {
			ov = append(ov, ovModel.OsName)
		}

		ovMap[ovModel.OsName] = append(ovMap[ovModel.OsName], ovModel.OsVersion)
	}

	c.deviceOsesMutex.Lock()
	c.deviceOsVerMap = ovMap
	c.deviceOses = ov
	c.deviceOsesMutex.Unlock()
	return nil
}

// GetAllDeviceMakes returns a slice of all devices brand_name capabilities in WM server
func (c *WmClient) GetAllDeviceMakes() ([]string, error) {

	err := c.loadDeviceMakesData()

	if err != nil && len(c.deviceMakes) > 0 {
		return nil, err
	}

	return c.deviceMakes, nil
}

// GetAllDevicesForMake returns a slice of an aggregate containing model_names and marketing_names for the given brand_name
func (c *WmClient) GetAllDevicesForMake(brandName string) ([]JSONModelMktName, error) {

	err := c.loadDeviceMakesData()

	if err != nil && len(c.deviceMakes) > 0 {
		return nil, err
	}
	c.deviceMakesMutex.Lock()
	if val, ok := c.deviceMakesMap[brandName]; ok {
		c.deviceMakesMutex.Unlock()
		return val, nil
	}
	c.deviceMakesMutex.Unlock()

	return nil, errors.New(fmt.Sprintf("Error getting data from WM server: %s does not exist", brandName))
}

func (c *WmClient) loadDeviceMakesData() error {
	// We lock the shared makeModel cache
	c.deviceMakesMutex.Lock()
	if c.deviceMakes != nil && len(c.deviceMakes) > 0 {
		defer c.deviceMakesMutex.Unlock()
		return nil
	}

	// if makeModel cache is empty unlock it
	c.deviceMakesMutex.Unlock()

	mkModels := make([]JSONMakeModel, 1000)
	var body, berr = c.internalGet("/v2/alldevices/json")
	if berr != nil {
		return berr
	}

	var merror = json.Unmarshal(body, &mkModels)
	if merror != nil {
		return merror
	}

	var dmMap = make(map[string][]JSONModelMktName, 0)
	var dm = make([]string, 0)

	for _, mkModel := range mkModels {
		if _, ok := dmMap[mkModel.BrandName]; !ok {
			dm = append(dm, mkModel.BrandName)
		}
		dmMap[mkModel.BrandName] = append(dmMap[mkModel.BrandName], JSONModelMktName{mkModel.ModelName, mkModel.MarketingName})
	}

	c.deviceMakesMutex.Lock()
	c.deviceMakesMap = dmMap
	c.deviceMakes = dm
	c.deviceMakesMutex.Unlock()
	return nil
}

// If given ltime is different from client internal one, all caches are cleared and client last load time is updated
func (c *WmClient) clearCachesIfNeeded(ltime string) {

	if len(ltime) > 0 && c.clientLtime != ltime {
		c.clientLtime = ltime
		c.clearCache()
	}
}

/*
 *
 * Project : WURFL Microservice 2.0 Client API
 *
 * Author(s): Paul Stephen Borile, Andrea Castello
 *
 * Date: Apr 23 2017 / September 2019
 *
 * Copyright (c) ScientiaMobile, Inc.
 * http://www.scientiamobile.com
 */
