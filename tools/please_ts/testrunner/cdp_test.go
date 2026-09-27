package testrunner

import (
	"encoding/json"
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
