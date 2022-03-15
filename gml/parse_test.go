package gml

import "testing"

func TestParse(t *testing.T) {
	t.Log(Parse(""))
	t.Fail()
}
