package discovery

import (
	"reflect"
	"testing"
)

func TestNewBBService(t *testing.T) {
	type args struct {
		endpoints []string
	}
	tests := []struct {
		name    string
		args    args
		want    BBEtcdDiscoveryInterface
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBBService(tt.args.endpoints)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBBService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBBService() got = %v, want %v", got, tt.want)
			}
		})
	}
}
