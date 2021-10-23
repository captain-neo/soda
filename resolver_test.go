package soda

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type SomeOtherType string
type embedStruct struct {
	A string `json:"a"`
	B time.Time
}
type SomeStruct struct {
	embedStruct
	Bool    bool
	Int     *int                      `json:"int" oai:"enum=1 2 6,default=10"`
	Int64   int64                     `json:"int64" oai:"description=我擦,maximum=100,minimum=-100"`
	Float64 float64                   `json:"float64"`
	String  string                    `json:"string" oai:"example=hello"`
	Bytes   []byte                    `json:"bytes"`
	JSON    json.RawMessage           `json:"json"`
	Time    time.Time                 `json:"time"`
	Slice   []SomeOtherType           `json:"slice"`
	Map     map[string]*SomeOtherType `json:"map"`

	Struct struct {
		X string `json:"x"`
	} `json:"struct"`

	EmptyStruct struct {
		Y string
	} `json:"structWithoutFields"`

	Ptr *SomeOtherType `json:"ptr"`
}

func TestResolveType(t *testing.T) {
	schema := ResolveModel(SomeStruct{})
	if err := schema.Validate(context.Background()); err != nil {
		t.Error(err)
	}
	b, err := schema.MarshalJSON()
	if err != nil {
		panic(err)
	}
	t.Log(string(b))
}
