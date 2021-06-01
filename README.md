[![Build Status][travis-image]][travis-url]
[![Go Report Card][goreportcard-image]][goreportcard-url]
[![codecov][codecov-image]][codecov-url]
[![Snap Status](https://build.snapcraft.io/badge/CanonicalLtd/iot-agent.svg)](https://build.snapcraft.io/user/CanonicalLtd/iot-agent)
# IoT Agent

The IoT Agent enrolls a device with the [IoT Identity](https://github.com/CanonicalLtd/iot-identity) service and 
receives credentials to access the MQTT broker. Via MQTT, it establishes communication with
an [IoT Management](https://github.com/CanonicalLtd/iot-management) service, so the device can be remotely monitored and managed over a
secure connection. The state of the device is mirrored in the cloud by the [IoT Device Twin](https://github.com/CanonicalLtd/iot-devicetwin) service.

The agent is intended to operate on a device running Ubuntu or Ubuntu Core with snapd enabled. 
The device management features are implemented using the snapd REST API.

  ![IoT Management Solution Overview](docs/IoTManagement.svg)
  

## Build
The project uses vendorized dependencies using govendor. Development has been done on minimum Go version 1.12.1.
```bash
$ go get github.com/CanonicalLtd/iot-agent
$ cd iot-agent
$ ./get-deps.sh
$ go build ./...
```

## daemons
### agent

The agent runs as a simple daemon in the snap. It uses the snap config value `nats.snapd.password` to protect
a partially implemented snapd NATS API that closely mirrors the snapd REST API. The details can be found in
the AsyncAPI [here](./asyncapi.yaml).

The NATS configuration currently needs to be setup separately on the server to support this.

## apps

### unregister

unregister is a separate app in the snap that can be used to clear the configuration of the agent related
to a Device Management Service it was previously registered/enrolled with. It will:

* Stop the agent
* Remove .secret
* Remove params
* Restart the agent

## Encrypted `snapd` subject tree

By default, the `snapd.>` subject tree is encrypted and only accessible with the correct username and password. By default (and with no snap configuration set) the password is `accept solve carbon atmosphere`. To change to a different password,

```bash
snap set everactive-iot-agent nats.snapd.password="a very long password indeed"
```
Note that this password must match what is set in `everactive-nats`.

## Unit Tests

Set `OVERRIDE_SNAP_DATA` and `OVERRIDE_SNAP_COMMON` to be values that are accessible / exists during testing. Make
sure you export the full path ex. `/tmp/test-output`. `${OVERRIDE_SNAP_DATA/current}` also needs to exist prior
to running tests.

### Run single test

`go test -v github.com/everactive/iot-agent/pkg/server -testify.m ^Test_NewServer$`


## Tasks

A top level Taskfile [taskfile.dev](https://taskfile.dev/#/) is included and drives common tasks. 

### initialize

If you haven't previously installed dependencies, this will install Mockery for you. It will then
execute the prebuild task.

### prebuild

Will do all necessary pre-build tasks that are not one-time (like regenerating mocks). Regenerating messages
structs is excluded do to manual editing necessary. See [generate-message-structs](#generate-message-structs) 
for more information.

### generate-message-structs

NOTE: The current pkg/messages/messages.go has to be hand edited to perserve some types. If you regenerate the file you will need to diff it to identify types
that are translated as string or float64 when their types are time.Time or int64. `import "time"` also needs to
be preserved and the empty interface generated is meaningless and is dropped.