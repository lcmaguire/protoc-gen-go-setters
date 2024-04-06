package tests

import (
	"testing"

	"github.com/lcmaguire/protoc-gen-go-setters/example"
)

func TestSetters(t *testing.T) {
	s := &example.SampleMessage{
		TestOneof: nil,
	}

	foo := &example.Foo{}
	s.SetFoo(foo)

	ex := example.Example{
		Sample: &example.SampleMessage{},
	}
	ex.GetSample().SetName("abcdefg")
	t.Log(ex.Sample)
}
