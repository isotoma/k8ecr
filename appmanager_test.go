package main

import (
	"reflect"
	"testing"
)

func TestGroupResources(t *testing.T) {
	tests := []struct {
		name        string
		deployments []Resource
		cronjobs    []Resource
		result      map[string]App
	}{
		{"empty", []Resource{}, []Resource{}, map[string]App{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupResources(tt.deployments, tt.cronjobs)
			if !reflect.DeepEqual(got, tt.result) {
				t.Errorf("groupResources got %v, want %v", got, tt.result)
			}
		})
	}
}
