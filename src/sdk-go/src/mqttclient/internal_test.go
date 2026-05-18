// Internal tests for the unexported helpers in the mqttclient package.
//
// They live in the same package (no _test suffix) so they can access
// parseServerURI and validateTopicWildcard, which are not part of the
// public SDK surface but are core to its correctness.
package mqttclient

import "testing"

func TestParseServerURI(t *testing.T) {
	cases := []struct {
		in        string
		wantProto string
		wantHost  string
		wantPort  int
	}{
		{"mqtt://broker:1883", "tcp", "broker", 1883},
		{"mqtt://broker", "tcp", "broker", 1883},
		{"mqtts://secure:8883", "ssl", "secure", 8883},
		{"mqtts://secure", "ssl", "secure", 8883},
		{"ws://broker:9001", "ws", "broker", 9001},
		{"wss://broker:9443", "wss", "broker", 9443},
		{"broker:1884", "tcp", "broker", 1884},
		{"broker", "tcp", "broker", 1883},
	}
	for _, tc := range cases {
		proto, host, port := parseServerURI(tc.in)
		if proto != tc.wantProto || host != tc.wantHost || port != tc.wantPort {
			t.Errorf("parseServerURI(%q) = (%q, %q, %d), want (%q, %q, %d)",
				tc.in, proto, host, port, tc.wantProto, tc.wantHost, tc.wantPort)
		}
	}
}

func TestValidateTopicWildcard(t *testing.T) {
	good := []string{"a/b", "a/b/#", "a/+/c", "#"}
	bad := []string{"a/#/b", "#/b", "a/#x"}

	for _, t1 := range good {
		if err := validateTopicWildcard(t1); err != nil {
			t.Errorf("expected %q to be valid, got: %v", t1, err)
		}
	}
	for _, t1 := range bad {
		if err := validateTopicWildcard(t1); err == nil {
			t.Errorf("expected %q to be invalid", t1)
		}
	}
}
