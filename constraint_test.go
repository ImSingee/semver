package semver

import (
	"reflect"
	"testing"
)

func TestParseConstraint(t *testing.T) {
	tests := []struct {
		in  string
		f   cfunc
		v   string
		err bool
	}{
		{">= 1.2", constraintGreaterThanEqual, "1.2", false},
		{"1.0", constraintTildeOrEqual, "1.0", false},
		{"foo", nil, "", true},
		{"<= 1.2", constraintLessThanEqual, "1.2", false},
		{"=< 1.2", constraintLessThanEqual, "1.2", false},
		{"=> 1.2", constraintGreaterThanEqual, "1.2", false},
		{"v1.2", constraintTildeOrEqual, "1.2", false},
		{"=1.5", constraintTildeOrEqual, "1.5", false},
		{"> 1.3", constraintGreaterThan, "1.3", false},
		{"< 1.4.1", constraintLessThan, "1.4.1", false},
		{"< 40.50.10", constraintLessThan, "40.50.10", false},
		{"0", constraintTildeOrEqual, "0", false},
		{"*", constraintTildeOrEqual, "0", false},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			c, err := parseConstraint(tc.in)
			if tc.err && err == nil {
				t.Fatalf("Expected error for %s didn't occur", tc.in)
			} else if !tc.err && err != nil {
				t.Fatalf("Unexpected error for %s: %s", tc.in, err)
			}

			// If an error was expected continue the loop and don't try the other
			// tests as they will cause errors.
			if tc.err {
				return
			}

			if tc.v != c.con.String() {
				t.Errorf("Incorrect version found on %s", tc.in)
			}

			f1 := reflect.ValueOf(tc.f)
			f2 := reflect.ValueOf(constraintOps[c.origfunc])
			if f1 != f2 {
				t.Errorf("Wrong constraint found for %s", tc.in)
			}
		})
	}
}

func TestConstraintNotEqual(t *testing.T) {
	cases := []struct {
		c    string
		v    string
		pass bool
	}{
		{"!=4.1.0", "4.1.0", false},
		{"!=4.1.0", "4.1.1", true},
		{"!=4.1", "4.1.0", false},
		{"!=4.1", "4.1.1", true},
		{"!=4.1", "5.1.0-alpha.1", true},
		{"!=4.1-alpha", "4.1.0", true},
		{"!=4.1", "5.1.0", true},
		{"!=4.1-alpha", "4.1.0-alpha", false},
		{"!=4.1-alpha", "4.1.1-alpha", true},
		{"!=4.1-alpha", "4.1.0", true},
		{"!=4.1", "5.1.0", true},
		{"!=4.x", "5.1.0", true},
		{"!=4.x", "4.1.0", false},
		{"!=4.1.x", "4.2.0", true},
		{"!=4.2.x", "4.2.3", false},
		{"!=4.2.x", "4-alpha", false},
		{"!=4.2.x", "4.2-alpha", false},
		{"!=4.2.x", "4.2.3-alpha", false},
		{"!=4.2.x-alpha", "4.2.3-alpha", false},
	}

	for _, tc := range cases {
		t.Run(tc.v+tc.c, func(t *testing.T) {
			v := MustParse(tc.v)
			c := mustParseConstraint(tc.c)

			pass, err := constraintNotEqual(v, c)

			if tc.pass && !pass {
				t.Fatalf("expect not equal but euqal: %s", err)
			}
			if !tc.pass && pass {
				t.Fatalf("expect not equal but report false")
			}
		})
	}
}

func TestConstraintGreaterThan(t *testing.T) {
	cases := []struct {
		c    string
		v    string
		pass bool
	}{
		{">1.1", "4.1.0", true},
		{">1.1", "1.1.0", false},
		{">0", "0", false},
		{">0", "1", true},
		{">0", "0.0.1-alpha", false},
		{">0.0", "0.0.1-alpha", false},
		{">0-0", "0.0.1-alpha", true},
		{">0.0-0", "0.0.1-alpha", true},
		{">0", "0.0.0-alpha", false},
		{">0-0", "0.0.0-alpha", true},
		{">0.0.0-0", "0.0.0-alpha", true},
		{">1.2.3-alpha.1", "1.2.3-alpha.2", true},
		{">1.2.3-alpha.1", "1.3.3-alpha.2", true},
		{">11", "11.1.0", true},
		{">11.1", "11.1.0", false},
		{">11.1", "11.1.1", true},
		{">11.1", "11.2.1", true},
	}

	for _, tc := range cases {
		t.Run(tc.v+tc.c, func(t *testing.T) {
			v := MustParse(tc.v)
			c := mustParseConstraint(tc.c)

			pass, err := constraintGreaterThan(v, c)

			if pass && !tc.pass {
				t.Fatalf("expect pass but not: %s", err)
			}
			if !pass && tc.pass {
				t.Fatalf("expect not pass but pass")
			}
		})
	}
}

func mustParseConstraint(c string) *constraint {
	cc, err := parseConstraint(c)
	if err != nil {
		panic(err)
	}
	return cc
}
