package bbeslog

import "testing"

func TestPushLogs(t *testing.T) {
	type args struct {
		logData   string
		addresses []string
		options   []EsLogOption
	}
	tests := []struct {
		name string
		args args
	}{
		{"one", args{logData: "error", addresses: []string{"http://127.0.0.1:9200"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
