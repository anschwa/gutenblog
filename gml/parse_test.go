package gml

import "testing"

func Test_parse(t *testing.T) {
	parse(gmlTestInput)
	t.Fail()
}
