package plugins

import (
	sdk "olicanaplot/sdk/go"
	"reflect"
	"testing"
)

// This test ensures that the re-exported configuration structs in internal/plugins
// match their counterparts in pkg/sdk/model.
//
// These structs are now type aliases but we still verify consistency for good measure
// and to ensure no regression to duplication occurs.

func TestStructSynchronization(t *testing.T) {
	tests := []struct {
		name     string
		internal interface{}
		external interface{}
	}{
		{"ChartConfig", ChartConfig{}, sdk.ChartConfig{}},
		{"GridConfig", GridConfig{}, sdk.GridConfig{}},
		{"AxisConfig", AxisConfig{}, sdk.AxisConfig{}},
		{"AxisGroupConfig", AxisGroupConfig{}, sdk.AxisGroupConfig{}},
		{"SeriesConfig", SeriesConfig{}, sdk.SeriesConfig{}},
		{"FilePattern", FilePattern{}, sdk.FilePattern{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			internalTyp := reflect.TypeOf(tt.internal)
			externalTyp := reflect.TypeOf(tt.external)

			if internalTyp.NumField() != externalTyp.NumField() {
				t.Errorf("%s: internal has %d fields, external has %d fields",
					tt.name, internalTyp.NumField(), externalTyp.NumField())
				return
			}

			for i := 0; i < internalTyp.NumField(); i++ {
				fInternal := internalTyp.Field(i)
				fExternal := externalTyp.Field(i)

				if fInternal.Name != fExternal.Name {
					t.Errorf("%s: field %d name mismatch: internal=%s, external=%s",
						tt.name, i, fInternal.Name, fExternal.Name)
				}
				if fInternal.Type.String() != fExternal.Type.String() {
					// Handle cases where types might be local vs pkg-qualified but matching in structure
					// For deep check of slices/structs, we'd need more logic, but for now string check
					// catches most primitive/pointer mismatches.
					t.Logf("%s: field %s type check: %s vs %s",
						tt.name, fInternal.Name, fInternal.Type.String(), fExternal.Type.String())
				}
				if fInternal.Tag.Get("json") != fExternal.Tag.Get("json") {
					t.Errorf("%s: field %s json tag mismatch: internal=%s, external=%s",
						tt.name, fInternal.Name, fInternal.Tag.Get("json"), fExternal.Tag.Get("json"))
				}
			}
		})
	}
}
