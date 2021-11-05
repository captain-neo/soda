package soda

import (
	"reflect"
	"testing"
)

func Test_parseStringSlice(t *testing.T) {
	type args struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			name: "ok with numeric",
			args: args{
				val: "1 2 3",
			},
			want: []interface{}{"1", "2", "3"},
		},
		{
			name: "ok with alphabets",
			args: args{
				val: "a b c",
			},
			want: []interface{}{"a", "b", "c"},
		},
		{
			name: "ok if single word",
			args: args{
				val: "abc",
			},
			want: []interface{}{"abc"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseStringSlice(tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseIntSlice(t *testing.T) {
	type args struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			name: "ok if some number",
			args: args{
				val: "1 2 3",
			},
			want: []interface{}{1, 2, 3},
		},
		{
			name: "ok if single number",
			args: args{
				val: "123",
			},
			want: []interface{}{123},
		},
		{
			name: "ignore non-number",
			args: args{
				val: "12 fake 23",
			},
			want: []interface{}{12, 23},
		},
		{
			name: "trim space",
			args: args{
				val: " 12 ",
			},
			want: []interface{}{12},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseIntSlice(tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseIntSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseFloatSlice(t *testing.T) {
	type args struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			name: "ok if some number",
			args: args{
				val: "1 2 3",
			},
			want: []interface{}{1.0, 2.0, 3.0},
		},
		{
			name: "ok if single number",
			args: args{
				val: "123",
			},
			want: []interface{}{123.0},
		},
		{
			name: "ignore non-number",
			args: args{
				val: "12 fake 23",
			},
			want: []interface{}{12.0, 23.0},
		},
		{
			name: "trim space",
			args: args{
				val: " 12 ",
			},
			want: []interface{}{12.0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseFloatSlice(tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseFloatSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toKebabCase(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "camel to kebab",
			args: args{
				str: "CamelCase",
			},
			want: "camel-case",
		},
		{
			name: "kebab to kebab",
			args: args{
				str: "camel-case",
			},
			want: "camel-case",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toKebabCase(tt.args.str); got != tt.want {
				t.Errorf("toKebabCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toCamelCase(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "kebab to camel",
			args: args{
				str: "camel-case",
			},
			want: "CamelCase",
		},
		{
			name: "camel to camel",
			args: args{
				str: "CamelCase",
			},
			want: "CamelCase",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toCamelCase(tt.args.str); got != tt.want {
				t.Errorf("toCamelCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getHandlerName(t *testing.T) {
	type args struct {
		fn interface{}
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test function",
			args: args{fn: Test_getHandlerName},
			want: "sodaTest_getHandlerName",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHandlerName(tt.args.fn); got != tt.want {
				t.Errorf("getHandlerName() = %v, want %v", got, tt.want)
			}
		})
	}
}
