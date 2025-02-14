# Changelog

### 2.1.3
- Updated dependencies

### 2.1.2
- Revamped example.go, fixed tests

### 2.1.1
- Fixed issue in `LookupHeaders(map[string]string` when correct headers are passed using mixed case keys (ie: "UsEr-Agent")

### 2.1.0
- Added `LookupHeaders(map[string]string` method to API

### 2.0.1
- Adapted unit tests to new server behavior of not returning error on empty user-agent or header. They are also adapted to run with different server configurations (ie AWS/Azure Professional edition, Docker server).

### 2.0.0
- Initial version: GetInfo, LookupUseragent, LookupRequest, LookupDeviceId and other functions to set cache and requested capabilitities
