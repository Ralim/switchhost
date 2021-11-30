package nca

import (
	"errors"
	"reflect"
	"testing"
)

func TestDecryptAes128Ecb(t *testing.T) {
	cases := []struct {
		desc        string
		input       []byte
		key         []byte
		expected    []byte
		expectederr error
	}{
		{"Test empty", []byte{}, []byte{}, []byte{}, nil},
		{"Test bad len", []byte{1, 2, 3}, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, []byte{0, 0, 0}, errors.New("invalid input length")},
		{"Test valid", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, []byte{185, 14, 186, 242, 222, 2, 144, 6, 161, 133, 58, 97, 27, 99, 109, 235}, nil},
	}
	for i, tc := range cases {
		actual, actualErr := decryptAes128Ecb(tc.input, tc.key)
		if !reflect.DeepEqual(tc.expected, actual) {
			t.Errorf("%d >%s: expected: >%v< got: >%v< for %+v/%+v", i,
				tc.desc, tc.expected, actual, tc.input, tc.key)
		}
		if tc.expectederr != nil && actualErr != nil {
			if tc.expectederr.Error() != actualErr.Error() {
				t.Errorf("%d >%s: expected: >%v< got: >%v< for %+v/%+v", i,
					tc.desc, tc.expectederr, actualErr, tc.input, tc.key)
			}
		} else if tc.expectederr != nil || actualErr != nil {
			t.Errorf("%d >%s: expected: >%v< got: >%v< for %+v/%+v", i,
				tc.desc, tc.expectederr, actualErr, tc.input, tc.key)
		}

	}
}
