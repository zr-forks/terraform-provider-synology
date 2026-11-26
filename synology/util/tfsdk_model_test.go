package util

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	addressAttrType = map[string]attr.Type{
		"street": types.StringType,
		"city":   types.StringType,
	}

	testType = map[string]attr.Type{
		"name":   types.StringType,
		"age":    types.Int64Type,
		"states": types.ListType{}.WithElementType(types.StringType),
		"addresses": types.ListType{}.WithElementType(
			types.ObjectType{}.WithAttributeTypes(addressAttrType),
		),
	}

	testValues = map[string]attr.Value{
		"name": types.StringValue("John"),
		"age":  types.Int64Value(30),
		"states": types.ListValueMust(
			types.StringType,
			[]attr.Value{types.StringValue("CA"), types.StringValue("NY")},
		),
		"addresses": types.ListValueMust(
			types.ObjectType{}.WithAttributeTypes(addressAttrType),
			[]attr.Value{
				types.ObjectValueMust(addressAttrType, map[string]attr.Value{
					"street": types.StringValue("123 Main St"),
					"city":   types.StringValue("San Francisco"),
				}),
			}),
	}
)

func TestGetType(t *testing.T) {
	type args struct {
		r any
	}
	tests := []struct {
		name    string
		args    args
		want    attr.Type
		wantErr bool
	}{
		{
			name: "TestGetType",
			args: args{
				r: struct {
					Name      string   `json:"name"`
					Age       int      `json:"age"`
					States    []string `json:"states"`
					Addresses []struct {
						Street string `json:"street"`
						City   string `json:"city"`
					} `json:"addresses"`
				}{},
			},
			want: types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
				"name":   types.StringType,
				"age":    types.Int64Type,
				"states": types.ListType{}.WithElementType(types.StringType),
				"addresses": types.ListType{}.WithElementType(
					types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
						"street": types.StringType,
						"city":   types.StringType,
					},
					),
				),
			}),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetType(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetValue(t *testing.T) {
	type args struct {
		r any
	}
	tests := []struct {
		name    string
		args    args
		want    attr.Value
		wantErr bool
	}{
		{
			name: "TestGetValue",
			args: args{
				r: struct {
					Name      string   `json:"name"`
					Age       int      `json:"age"`
					States    []string `json:"states"`
					Addresses []struct {
						Street string `json:"street"`
						City   string `json:"city"`
					} `json:"addresses"`
				}{
					Name:   "John",
					Age:    30,
					States: []string{"CA", "NY"},
					Addresses: []struct {
						Street string `json:"street"`
						City   string `json:"city"`
					}{
						{
							Street: "123 Main St",
							City:   "San Francisco",
						},
					},
				},
			},
			want:    types.ObjectValueMust(testType, testValues),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetValue(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// DSM 6.2 storage load_info replies contain heterogeneous nested maps
// (env, can_do, vspace_can_do, ...). These used to panic in mapValue:
// nested maps were dropped (missing attribute) and, once included, their
// concrete Object values were rejected by DynamicType attributes.
func TestGetValue_NestedMaps(t *testing.T) {
	in := map[string]any{
		"name": "pool",
		"env": map[string]any{
			"data_scrubbing": map[string]any{
				"sche_enabled": "true",
				"sche_status":  "idle",
			},
			"isns": map[string]any{
				"enabled": true,
				"address": "10.0.0.1",
			},
		},
		"can_do": map[string]any{
			"delete": true,
		},
		"disks": []any{"sda", "sdb"},
		"empty": nil,
	}

	v, err := GetValue(in)
	if err != nil {
		t.Fatalf("GetValue: %v", err)
	}
	obj, ok := v.(types.Object)
	if !ok {
		t.Fatalf("got %T, want types.Object", v)
	}
	attrs := obj.Attributes()
	for _, key := range []string{"name", "env", "can_do", "disks"} {
		if _, ok := attrs[key]; !ok {
			t.Errorf("attribute %q missing from result", key)
		}
	}
	// nested map must be preserved as a Dynamic value carrying an Object
	env, ok := attrs["env"].(types.Dynamic)
	if !ok {
		t.Fatalf("env has type %T, want types.Dynamic", attrs["env"])
	}
	inner, ok := env.UnderlyingValue().(types.Object)
	if !ok {
		t.Errorf("env inner value has type %T, want types.Object", env.UnderlyingValue())
	} else if _, ok := inner.Attributes()["data_scrubbing"]; !ok {
		t.Error("env.data_scrubbing missing")
	}
}
