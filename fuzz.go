//go:build go1.18
// +build go1.18

package semver

import "testing"

func FuzzNewVersion(f *testing.F) {
	testcases := []string{"v1.2.3", " ", "......", "1", "1.2.3-beta.1", "1.2.3+foo", "2.3.4-alpha.1+bar", "lorem ipsum"}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, a string) {
		_, _ = NewVersion(a)
	})
}

func FuzzStrictNewVersion(f *testing.F) {
	testcases := []string{"v1.2.3", " ", "......", "1", "1.2.3-beta.1", "1.2.3+foo", "2.3.4-alpha.1+bar", "lorem ipsum"}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, a string) {
		_, _ = StrictNewVersion(a)
	})
}

func FuzzNewConstraint(f *testing.F) {
	testcases := []string{
		"v1.2.3",
		" ",
		"......",
		"1",
		"1.2.3-beta.1",
		"1.2.3+foo",
		"2.3.4-alpha.1+bar",
		"lorem ipsum",
		"*",
		"!=1.2.3",
		"^4.5",
		"1.0.0 - 2",
		"1.2.3.4.5.6",
		">= 1",
		"~9.8.7",
		"<= 12.13.14",
		"987654321.123456789.654123789",
		"1.x",
		"2.3.x",
		"9.2-beta.0",
	}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, a string) {
		_, _ = NewConstraint(a)
	})
}
