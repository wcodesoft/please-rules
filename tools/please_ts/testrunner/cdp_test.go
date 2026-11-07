package testrunner

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCDPMessageSerializationMatrix(t *testing.T) {
	tests := []struct {
		name      string
		msg       cdpMessage
		wantID    int64
		wantMeth  string
		hasParams bool
	}{
		{
			name: "enable runtime method without params",
			msg: cdpMessage{
				ID:     1,
				Method: "Runtime.enable",
			},
			wantID:    1,
			wantMeth:  "Runtime.enable",
			hasParams: false,
		},
		{
			name: "evaluate method with params",
			msg: cdpMessage{
				ID:     2,
				Method: "Runtime.evaluate",
				Params: map[string]interface{}{
					"expression": "1 + 1",
				},
			},
			wantID:    2,
			wantMeth:  "Runtime.evaluate",
			hasParams: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("failed marshaling cdpMessage: %v", err)
			}

			var decoded cdpMessage
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed unmarshaling cdpMessage: %v", err)
			}

			if decoded.ID != tt.wantID {
				t.Errorf("decoded.ID = %d, want %d", decoded.ID, tt.wantID)
			}
			if decoded.Method != tt.wantMeth {
				t.Errorf("decoded.Method = %q, want %q", decoded.Method, tt.wantMeth)
			}
			if (decoded.Params != nil) != tt.hasParams {
				t.Errorf("decoded.Params presence = %v, want %v", decoded.Params != nil, tt.hasParams)
			}
		})
	}
}

func TestCDPDispatchRoutesEventsAndResponses(t *testing.T) {
	c := &CDPClient{pending: make(map[int64]chan cdpResponse)}

	var got []string
	c.OnEvent("Runtime.bindingCalled", func(params json.RawMessage) { got = append(got, "a:"+string(params)) })
	c.OnEvent("Runtime.bindingCalled", func(params json.RawMessage) { got = append(got, "b") })
	c.OnEvent("Page.loadEventFired", func(json.RawMessage) { got = append(got, "load") })

	c.dispatch([]byte(`{"method":"Runtime.bindingCalled","params":{"name":"x"}}`))
	if want := `a:{"name":"x"},b`; strings.Join(got, ",") != want {
		t.Errorf("event handlers ran as %q, want %q", strings.Join(got, ","), want)
	}

	// An event nobody registered for, and a frame that is neither event nor response, are ignored.
	c.dispatch([]byte(`{"method":"Network.requestWillBeSent","params":{}}`))
	c.dispatch([]byte(`{"foo":1}`))
	c.dispatch([]byte(`not json`))
	if len(got) != 2 {
		t.Errorf("unregistered or malformed frames reached a handler: %v", got)
	}

	ok := make(chan cdpResponse, 1)
	bad := make(chan cdpResponse, 1)
	c.pending[7], c.pending[8] = ok, bad
	c.dispatch([]byte(`{"id":7,"result":{"data":"abc"}}`))
	c.dispatch([]byte(`{"id":8,"error":{"code":-32000,"message":"nope"}}`))
	if r := <-ok; r.Error != nil || string(r.Result) != `{"data":"abc"}` {
		t.Errorf("response 7 = %+v", r)
	}
	if r := <-bad; r.Error == nil || !strings.Contains(r.Error.Error(), "nope") {
		t.Errorf("response 8 = %+v, want the CDP error", r)
	}
	if len(c.pending) != 0 {
		t.Errorf("completed commands left pending: %v", c.pending)
	}
}
