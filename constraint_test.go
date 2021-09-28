package semver

import (
	"testing"
)

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
