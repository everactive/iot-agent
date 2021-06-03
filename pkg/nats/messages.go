package nats

import (
	"github.com/snapcore/snapd/client"
)

const (
	// These are unversioned, existed pre-AsyncAPI specification
	AssertionGetSubject = "snapd.v2.assertions.get"
	SnapsGetSubject     = "snapd.v2.snaps.get"
	AppsPostSubject     = "snapd.v2.apps.post"

	// These are versioned and exist in the AsyncAPI specification
	AssertionGetSubjectv1  = "v1.snapd.v2.assertions.get"
	SnapsGetSubjectv1      = "v1.snapd.v2.snaps.get"
	SnapsSnapPostSubjectv1 = "v1.snapd.v2.snaps.*.post"
	AppsPostSubjectv1      = "v1.snapd.v2.apps.post"

	IotAgentMqttBrokerConnected = "iot.agent.mqtt.connection.status"
)

type emptyMessage struct{}

// AssertionsRequest is the request for an assertions stream based on the type specified.
// Reference: https://snapcraft.io/docs/snapd-api#heading--assertions for types
type AssertionsRequest struct {
	Type string `json:"type"`
}

// AssertionsResponse is a response containing the assertions based on the type requested as
// specified in the AssertionsRequest. It will be a stream of assertions, each stream is potentially
// multiple assertions. The are separated by double new lines.
// Reference: https://snapcraft.io/docs/snapd-api#heading--assertions
type AssertionsResponse struct {
	Stream string `json:"stream"`
}

// SnapsResponse is the response for getting a list of snaps. A request is made with the empty message.
// Reference: https://snapcraft.io/docs/snapd-api#heading--snaps
type SnapsResponse struct {
	Snaps []client.Snap `json:"snaps"`
}

// AppsRequest is the message to act on snaps or their services
// Reference: https://snapcraft.io/docs/snapd-api#heading--apps
type AppsRequest struct {
	Action string   `json:"action"`
	Names  []string `json:"names"`
}

// AppsResponse is the response to an AppsRequest, contains the change ID or an error message
type AppsResponse struct {
	ChangeID string  `json:"changeId"`
	Error    *string `json:"error,omitempty"`
}
