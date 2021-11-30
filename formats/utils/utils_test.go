package utils

import (
	"testing"
)

func TestCString(t *testing.T) {
	cases := []struct {
		desc     string
		input    []byte
		expected string
	}{
		{"Test empty", []byte{}, ""},
		{"Test null", []byte{0}, ""},
		{"Test normal", []byte{65, 67, 68, 0}, "ACD"},
		{"Test no term", []byte{65, 67, 68}, "ACD"},
		{"Test middle null", []byte{65, 67, 68, 0, 60, 61}, "ACD"},
	}
	for i, tc := range cases {
		actual := CString(tc.input)
		if actual != tc.expected {
			t.Errorf("%d >%s: expected: >%s< got: >%s< for %+v", i,
				tc.desc, tc.expected, actual, tc.input)
		}
	}
}
