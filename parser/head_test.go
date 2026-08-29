package parser

import (
	"reflect"
	"strings"
	"testing"

	"excel2pb/config"
)

func TestConfigureFiltersUsesOnlyConfiguredTargets(t *testing.T) {
	previous := append([]string(nil), AllFilters...)
	t.Cleanup(func() { AllFilters = previous })

	tests := []struct {
		name string
		outs map[string]config.OutConfig
		want []string
	}{
		{name: "client only", outs: map[string]config.OutConfig{"Client": {}}, want: []string{ClientFlag}},
		{name: "server only", outs: map[string]config.OutConfig{"Server": {}}, want: []string{ServerFlag}},
		{name: "both", outs: map[string]config.OutConfig{"Server": {}, "Client": {}}, want: []string{ClientFlag, ServerFlag}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ConfigureFilters(test.outs); err != nil {
				t.Fatalf("ConfigureFilters() error = %v", err)
			}
			if !reflect.DeepEqual(AllFilters, test.want) {
				t.Fatalf("AllFilters = %v, want %v", AllFilters, test.want)
			}
		})
	}
}

func TestConfigureFiltersRejectsMissingTargets(t *testing.T) {
	previous := append([]string(nil), AllFilters...)
	t.Cleanup(func() { AllFilters = previous })

	err := ConfigureFilters(map[string]config.OutConfig{})
	if err == nil || !strings.Contains(err.Error(), "at least Client or Server") {
		t.Fatalf("ConfigureFilters() error = %v", err)
	}
}
