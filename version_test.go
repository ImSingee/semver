package semver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ImSingee/tt"
)

var strictVersionPass = []string{
	"1.2.3",
	"1.2.3+test.01",
	"1.2.3-alpha.-1",
	"1.0",
	"1",
	"1.2-5",
	"1.2-beta.5",
	"1.2.0-x.Y.0+metadata",
	"1.2.0-x.Y.0+metadata-width-hypen",
	"1.2.3-rc1-with-hypen",
	"1.2.3.4",
	"1.2.2147483648",
	"1.2147483648.3",
	"2147483648.3.0",
}

var softVersionFail = []string{
	"1.2.3-alpha.01",
	"1.2.beta",
	"v1.2.beta",
	"foo",
	"\n1.2",
	"\nv1.2",
	".",
	"1.",
	".1",
}

func TestStrictNewVersion(t *testing.T) {
	for _, v := range strictVersionPass {
		t.Run(v, func(t *testing.T) {
			_, err := StrictNewVersion(v)
			if err != nil {
				t.Fatalf("Error: %s", err)
			}
		})
	}

	for _, v := range softVersionFail {
		t.Run(v, func(t *testing.T) {
			_, err := StrictNewVersion(v)
			if err == nil {
				t.Fatal("Expect Error")
			}
		})
	}
}

func TestNewVersion(t *testing.T) {
	for _, v := range strictVersionPass {
		t.Run(v, func(t *testing.T) {
			_, err := NewVersion(v)
			if err != nil {
				t.Fatalf("Error: %s", err)
			}
		})

		t.Run("v"+v, func(t *testing.T) {
			_, err := NewVersion("v" + v)
			if err != nil {
				t.Fatalf("Error: %s", err)
			}
		})
	}

	for _, v := range softVersionFail {
		t.Run(v, func(t *testing.T) {
			_, err := NewVersion(v)
			if err == nil {
				t.Fatal("Expect Error")
			}
		})
	}
}

func TestNewVersionByParts(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		v := NewVersionByParts()
		tt.AssertIsNotNil(t, v)
		tt.AssertEqual(t, 0, len(v.parts))
		tt.AssertEqual(t, "", v.original)
	})

	t.Run("standard", func(t *testing.T) {
		v := NewVersionByParts(1, 2, 13)
		tt.AssertIsNotNil(t, v)
		tt.AssertEqual(t, []uint64{1, 2, 13}, v.parts)
		tt.AssertEqual(t, "1.2.13", v.original)
	})
}

func TestOriginal(t *testing.T) {
	tests := []string{
		"1.2.3",
		"v1.2.3",
		"1.0",
		"v1.0",
		"1",
		"v1",
		"1.2-5",
		"v1.2-5",
		"1.2-beta.5",
		"v1.2-beta.5",
		"1.2.0-x.Y.0+metadata",
		"v1.2.0-x.Y.0+metadata",
		"1.2.0-x.Y.0+metadata-width-hypen",
		"v1.2.0-x.Y.0+metadata-width-hypen",
		"1.2.3-rc1-with-hypen",
		"v1.2.3-rc1-with-hypen",
	}

	for _, tc := range tests {
		v, err := NewVersion(tc)
		if err != nil {
			t.Errorf("Error parsing version %s", tc)
		}

		o := v.Original()
		if o != tc {
			t.Errorf("Error retrieving original. Expected '%s' but got '%v'", tc, v)
		}
	}
}

func TestParts(t *testing.T) {
	v, err := NewVersion("1.2.3-beta.1+build.123")
	if err != nil {
		t.Error("Error parsing version 1.2.3-beta.1+build.123")
	}

	if v.PartsNumber() != 3 {
		t.Error("PartsNumber() returning wrong value")
	}
	if v.Major() != 1 || v.Part(1) != 1 {
		t.Error("Major() | Part(1) returning wrong value")
	}
	if v.Minor() != 2 || v.Part(2) != 2 {
		t.Error("Minor() | Part(2) returning wrong value")
	}
	if v.Patch() != 3 || v.Part(3) != 3 {
		t.Error("Patch() | Part(3) returning wrong value")
	}
	if v.Part(4) != 0 {
		t.Error("Part(4) returning wrong value")
	}
	if v.Prerelease() != "beta.1" {
		t.Error("Prerelease() returning wrong value")
	}
	if v.Metadata() != "build.123" {
		t.Error("Metadata() returning wrong value")
	}
}

func TestCoerceString(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"1.2.3", "1.2.3"},
		{"v1.2.3", "1.2.3"},
		{"1.0", "1.0"},
		{"v1.0", "1.0"},
		{"1", "1"},
		{"v1", "1"},
		{"1.2-5", "1.2-5"},
		{"v1.2-5", "1.2-5"},
		{"1.2-beta.5", "1.2-beta.5"},
		{"v1.2-beta.5", "1.2-beta.5"},
		{"1.2.0-x.Y.0+metadata", "1.2.0-x.Y.0+metadata"},
		{"v1.2.0-x.Y.0+metadata", "1.2.0-x.Y.0+metadata"},
		{"1.2.0-x.Y.0+metadata-width-hypen", "1.2.0-x.Y.0+metadata-width-hypen"},
		{"v1.2.0-x.Y.0+metadata-width-hypen", "1.2.0-x.Y.0+metadata-width-hypen"},
		{"1.2.3-rc1-with-hypen", "1.2.3-rc1-with-hypen"},
		{"v1.2.3-rc1-with-hypen", "1.2.3-rc1-with-hypen"},
	}

	for _, tc := range tests {
		v, err := NewVersion(tc.version)
		if err != nil {
			t.Errorf("Error parsing version %s", tc)
		}

		s := v.String()
		if s != tc.expected {
			t.Errorf("Error generating string. Expected '%s' but got '%s'", tc.expected, s)
		}
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int // -1 for <, 0 for =, 1 for >
	}{
		{"1.2.3", "1.5.1", -1},
		{"2.2.3", "1.5.1", 1},
		{"2.2.3", "2.2.2", 1},
		{"3.2-beta", "3.2-beta", 0},
		{"1.3", "1.1.4", 1},
		{"4.2", "4.2-beta", 1}, // 4.2 > 4.2-beta
		{"4.2-beta", "4.2", -1},
		{"4.2-alpha", "4.2-beta", -1}, // 4.2-alpha < 4.2-beta
		{"4.2-alpha", "4.2-alpha", 0},
		{"4.2-alpha", "4.2-alpha.1", -1},    // 4.2-alpha < 4.2-alpha.1
		{"4.2-alpha.1", "4.2-alpha.2", -1},  // 4.2-alpha.1 < 4.2-alpha.2
		{"4.2-alpha.2", "4.2-alpha.9", -1},  // 4.2-alpha.2 < 4.2-alpha.9
		{"4.2-alpha.9", "4.2-alpha.10", -1}, // 4.2-alpha.9 < 4.2-alpha.10
		{"4.2-alpha.1", "4.2-alpha.10", -1}, // 4.2-alpha.1 < 4.2-alpha.10
		{"4.2-alpha.2", "4.2-alpha.11", -1}, // 4.2-alpha.2 < 4.2-alpha.11
		{"4.2-alpha.999", "4.2-beta", -1},   // 4.2-alpha.999 < 4.2-beta
		{"4.2-alpha.999", "4.2-beta.1", -1}, // 4.2-alpha.999 < 4.2-beta.1
		{"4.2-beta.2", "4.2-beta.1", 1},
		{"4.2-beta2", "4.2-beta1", 1},
		{"4.2-beta", "4.2-beta.2", -1},
		{"4.2-beta", "4.2-beta.foo", -1},
		{"4.2-beta.2", "4.2-beta", 1},
		{"4.2-beta.foo", "4.2-beta", 1},
		{"1.2+bar", "1.2+baz", 0},
		{"1.0.0-beta.4", "1.0.0-beta.-2", -1},
		{"1.0.0-beta.-2", "1.0.0-beta.-3", -1},
		{"1.0.0-beta.-3", "1.0.0-beta.5", 1},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		v2, err := NewVersion(tc.v2)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		a := v1.Compare(v2)
		b := v2.Compare(v1)

		if a != tc.expected {
			t.Errorf(
				"Comparison of '%s' and '%s' failed. Expected '%d', got '%d'",
				tc.v1, tc.v2, tc.expected, a,
			)
		}
		if b != -tc.expected {
			t.Errorf(
				"Comparison of '%s' and '%s' failed. Expected '%d', got '%d'",
				tc.v2, tc.v1, -tc.expected, b,
			)
		}
	}
}

func TestLessThan(t *testing.T) {
	tests := []struct {
		v1    string
		v2    string
		equal bool
	}{
		{"1.2.3", "1.5.1", false},
		{"1.5.1", "2.2.3", false},
		{"3.2-beta", "3.2-beta", true},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		v2, err := NewVersion(tc.v2)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		a := v1.LessThan(v2)
		b := v2.LessThan(v1)

		if tc.equal { // a b should be false
			if a != false {
				t.Errorf("'%s' < '%s' should be false", tc.v1, tc.v2)
			}
			if b != false {
				t.Errorf("'%s' < '%s' should be false", tc.v2, tc.v1)
			}
		} else { // a should be true while b should be false
			if a != true {
				t.Errorf("'%s' < '%s' should be true", tc.v1, tc.v2)
			}
			if b != false {
				t.Errorf("'%s' < '%s' should be false", tc.v2, tc.v1)
			}
		}
	}
}

func TestGreaterThan(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.2.3", "1.5.1", false},
		{"2.2.3", "1.5.1", true},
		{"3.2-beta", "3.2-beta", false},
		{"3.2.0-beta.1", "3.2.0-beta.5", false},
		{"3.2-beta.4", "3.2-beta.2", true},
		{"7.43.0-SNAPSHOT.99", "7.43.0-SNAPSHOT.103", false},
		{"7.43.0-SNAPSHOT.FOO", "7.43.0-SNAPSHOT.103", true},
		{"7.43.0-SNAPSHOT.99", "7.43.0-SNAPSHOT.BAR", false},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		v2, err := NewVersion(tc.v2)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		a := v1.GreaterThan(v2)
		e := tc.expected
		if a != e {
			t.Errorf(
				"Comparison of '%s' and '%s' failed. Expected '%t', got '%t'",
				tc.v1, tc.v2, e, a,
			)
		}
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1", "1.0", true},
		{"1.0", "1.0.0", true},
		{"1.0", "1.0.1", false},
		{"1.2.3", "1.5.1", false},
		{"2.2.3", "1.5.1", false},
		{"3.2-beta", "3.2-beta", true},
		{"3.2-beta+foo", "3.2-beta+bar", true},
		{"1.0", "1.0.0+foo", true},
		{"1.0+foo", "1.0.0+bar", true},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		v2, err := NewVersion(tc.v2)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		a := v1.Equal(v2)
		e := tc.expected
		if a != e {
			t.Errorf(
				"Comparison of '%s' and '%s' failed. Expected '%t', got '%t'",
				tc.v1, tc.v2, e, a,
			)
		}
	}
}

func TestIncPart(t *testing.T) {
	tests := []struct {
		v1               string
		expected         string
		part             int
		expectedOriginal string
	}{
		{"1.2.3.4", "1.2.3.5", 4, "1.2.3.5"},
		{"1.2.3.4", "1.2.4.0", 3, "1.2.4.0"},
		{"1.2.3", "1.2.4", 3, "1.2.4"},
		{"v1.2.4", "1.2.5", 3, "v1.2.5"},
		{"1.2.3", "1.3.0", 2, "1.3.0"},
		{"1.2", "1.2.0.1", 4, "1.2.0.1"},
		{"1.2", "1.2.1", 3, "1.2.1"},
		{"v1.2.4", "1.3.0", 2, "v1.3.0"},
		{"1.2.3", "2.0.0", 1, "2.0.0"},
		{"v1.2.4", "2.0.0", 1, "v2.0.0"},
		{"1.2.3+meta", "1.2.3.1", 4, "1.2.3.1"},
		{"1.2.3+meta", "1.2.4", 3, "1.2.4"},
		{"1.2.3-beta+meta", "1.2.3.0", 4, "1.2.3.0"},
		{"1.2.3-beta+meta", "1.2.3", 3, "1.2.3"},
		{"v1.2.4-beta+meta", "1.2.4", 3, "v1.2.4"},
		{"1.2.3-beta+meta", "1.3.0", 2, "1.3.0"},
		{"v1.2.4-beta+meta", "1.3.0", 2, "v1.3.0"},
		{"1.2.3-beta+meta", "2.0.0", 1, "2.0.0"},
		{"v1.2.4-beta+meta", "2.0.0", 1, "v2.0.0"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s-increase-%d", tc.v1, tc.part), func(t *testing.T) {
			v1, err := NewVersion(tc.v1)
			if err != nil {
				t.Errorf("Error parsing version: %s", err)
			}
			v2 := v1.IncPart(tc.part)

			a := v2.String()
			e := tc.expected
			if a != e {
				t.Errorf(
					"IncPart %d failed. Expected %q got %q",
					tc.part, e, a,
				)
			}

			a = v2.Original()
			e = tc.expectedOriginal
			if a != e {
				t.Errorf(
					"IncPart %d failed. Expected original %q got %q",
					tc.part, e, a,
				)
			}
		})
	}

}

func TestInc(t *testing.T) {
	tests := []struct {
		v1               string
		expected         string
		how              string
		expectedOriginal string
	}{
		{"1.2.3.4", "1.2.3.5", "last", "1.2.3.5"},
		{"1.2.3", "1.2.4", "last", "1.2.4"},
		{"v1.2.4", "1.2.5", "last", "v1.2.5"},
		{"1.2.3", "1.2.4", "patch", "1.2.4"},
		{"v1.2.4", "1.2.5", "patch", "v1.2.5"},
		{"1.2.3", "1.3.0", "minor", "1.3.0"},
		{"v1.2.4", "1.3.0", "minor", "v1.3.0"},
		{"1.2.3", "2.0.0", "major", "2.0.0"},
		{"v1.2.4", "2.0.0", "major", "v2.0.0"},
		{"1.2.3+meta", "1.2.4", "last", "1.2.4"},
		{"1.2.3+meta", "1.2.4", "patch", "1.2.4"},
		{"1.2.3-beta+meta", "1.2.3", "last", "1.2.3"},
		{"1.2.3-beta+meta", "1.2.3", "patch", "1.2.3"},
		{"v1.2.4-beta+meta", "1.2.4", "patch", "v1.2.4"},
		{"1.2.3-beta+meta", "1.3.0", "minor", "1.3.0"},
		{"v1.2.4-beta+meta", "1.3.0", "minor", "v1.3.0"},
		{"1.2.3-beta+meta", "2.0.0", "major", "2.0.0"},
		{"v1.2.4-beta+meta", "2.0.0", "major", "v2.0.0"},
	}

	for _, tc := range tests {
		t.Run(tc.v1+"-increase-"+tc.how, func(t *testing.T) {
			v1, err := NewVersion(tc.v1)
			if err != nil {
				t.Errorf("Error parsing version: %s", err)
			}
			var v2 Version
			switch tc.how {
			case "patch":
				v2 = v1.IncPatch()
			case "minor":
				v2 = v1.IncMinor()
			case "major":
				v2 = v1.IncMajor()
			case "last":
				v2 = v1.IncLast()
			}

			a := v2.String()
			e := tc.expected
			if a != e {
				t.Errorf(
					"Inc %q failed. Expected %q got %q",
					tc.how, e, a,
				)
			}

			a = v2.Original()
			e = tc.expectedOriginal
			if a != e {
				t.Errorf(
					"Inc %q failed. Expected original %q got %q",
					tc.how, e, a,
				)
			}
		})
	}
}

func TestSetPrerelease(t *testing.T) {
	tests := []struct {
		v1                 string
		prerelease         string
		expectedVersion    string
		expectedPrerelease string
		expectedOriginal   string
		expectedErr        error
	}{
		{"1.2.3", "**", "1.2.3", "", "1.2.3", ErrInvalidPrerelease},
		{"1.2.3", "beta", "1.2.3-beta", "beta", "1.2.3-beta", nil},
		{"v1.2.4", "beta", "1.2.4-beta", "beta", "v1.2.4-beta", nil},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		v2, err := v1.SetPrerelease(tc.prerelease)
		if err != tc.expectedErr {
			t.Errorf("Expected to get err=%s, but got err=%s", tc.expectedErr, err)
		}

		a := v2.Prerelease()
		e := tc.expectedPrerelease
		if a != e {
			t.Errorf("Expected prerelease value=%q, but got %q", e, a)
		}

		a = v2.String()
		e = tc.expectedVersion
		if a != e {
			t.Errorf("Expected version string=%q, but got %q", e, a)
		}

		a = v2.Original()
		e = tc.expectedOriginal
		if a != e {
			t.Errorf("Expected version original=%q, but got %q", e, a)
		}
	}
}

func TestSetMetadata(t *testing.T) {
	tests := []struct {
		v1               string
		metadata         string
		expectedVersion  string
		expectedMetadata string
		expectedOriginal string
		expectedErr      error
	}{
		{"1.2.3", "**", "1.2.3", "", "1.2.3", ErrInvalidMetadata},
		{"1.2.3", "meta", "1.2.3+meta", "meta", "1.2.3+meta", nil},
		{"v1.2.4", "meta", "1.2.4+meta", "meta", "v1.2.4+meta", nil},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		if err != nil {
			t.Errorf("Error parsing version: %s", err)
		}

		v2, err := v1.SetMetadata(tc.metadata)
		if err != tc.expectedErr {
			t.Errorf("Expected to get err=%s, but got err=%s", tc.expectedErr, err)
		}

		a := v2.Metadata()
		e := tc.expectedMetadata
		if a != e {
			t.Errorf("Expected metadata value=%q, but got %q", e, a)
		}

		a = v2.String()
		e = tc.expectedVersion
		if e != a {
			t.Errorf("Expected version string=%q, but got %q", e, a)
		}

		a = v2.Original()
		e = tc.expectedOriginal
		if a != e {
			t.Errorf("Expected version original=%q, but got %q", e, a)
		}
	}
}

func TestOriginalVPrefix(t *testing.T) {
	tests := []struct {
		version string
		vprefix string
	}{
		{"1.2.3", ""},
		{"v1.2.4", "v"},
	}

	for _, tc := range tests {
		v1, _ := NewVersion(tc.version)
		a := v1.originalVPrefix()
		e := tc.vprefix
		if a != e {
			t.Errorf("Expected vprefix=%q, but got %q", e, a)
		}
	}
}

func TestJsonMarshal(t *testing.T) {
	sVer := "1.1.1"
	x, err := StrictNewVersion(sVer)
	if err != nil {
		t.Errorf("Error creating version: %s", err)
	}
	out, err2 := json.Marshal(x)
	if err2 != nil {
		t.Errorf("Error marshaling version: %s", err2)
	}
	got := string(out)
	want := fmt.Sprintf("%q", sVer)
	if got != want {
		t.Errorf("Error marshaling unexpected marshaled content: got=%q want=%q", got, want)
	}
}

func TestJsonUnmarshal(t *testing.T) {
	sVer := "1.1.1"
	ver := &Version{}
	err := json.Unmarshal([]byte(fmt.Sprintf("%q", sVer)), ver)
	if err != nil {
		t.Errorf("Error unmarshaling version: %s", err)
	}
	got := ver.String()
	want := sVer
	if got != want {
		t.Errorf("Error unmarshaling unexpected object content: got=%q want=%q", got, want)
	}
}

func TestSQLScanner(t *testing.T) {
	sVer := "1.1.1"
	x, err := StrictNewVersion(sVer)
	if err != nil {
		t.Errorf("Error creating version: %s", err)
	}
	var s sql.Scanner = x
	var out *Version
	var ok bool
	if out, ok = s.(*Version); !ok {
		t.Errorf("Error expected Version type, got=%T want=%T", s, Version{})
	}
	got := out.String()
	want := sVer
	if got != want {
		t.Errorf("Error sql scanner unexpected scan content: got=%q want=%q", got, want)
	}
}

func TestDriverValuer(t *testing.T) {
	sVer := "1.1.1"
	x, err := StrictNewVersion(sVer)
	if err != nil {
		t.Errorf("Error creating version: %s", err)
	}
	got, err := x.Value()
	if err != nil {
		t.Fatalf("Error getting value, got %v", err)
	}
	want := sVer
	if got != want {
		t.Errorf("Error driver valuer unexpected value content: got=%q want=%q", got, want)
	}
}

func TestValidatePrerelease(t *testing.T) {
	tests := []struct {
		pre      string
		expected error
	}{
		{"foo", nil},
		{"alpha.1", nil},
		{"alpha.01", ErrSegmentStartsZero},
		{"foo☃︎", ErrInvalidPrerelease},
		{"alpha.0-1", nil},
	}

	for _, tc := range tests {
		if err := validatePrerelease(tc.pre); err != tc.expected {
			t.Errorf("Unexpected error %q for prerelease %q", err, tc.pre)
		}
	}
}

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		meta     string
		expected error
	}{
		{"foo", nil},
		{"alpha.1", nil},
		{"alpha.01", nil},
		{"foo☃︎", ErrInvalidMetadata},
		{"alpha.0-1", nil},
		{"al-pha.1Phe70CgWe050H9K1mJwRUqTNQXZRERwLOEg37wpXUb4JgzgaD5YkL52ABnoyiE", nil},
	}

	for _, tc := range tests {
		if err := validateMetadata(tc.meta); err != tc.expected {
			t.Errorf("Unexpected error %q for metadata %q", err, tc.meta)
		}
	}
}

func benchNewVersion(v string, b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewVersion(v)
	}
}

func benchStrictNewVersion(v string, b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = StrictNewVersion(v)
	}
}

func BenchmarkNewVersionSimple(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	benchNewVersion("1.0.0", b)
}

func BenchmarkCoerceNewVersionSimple(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	benchStrictNewVersion("1.0.0", b)
}

func BenchmarkNewVersionPre(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	benchNewVersion("1.0.0-alpha", b)
}

func BenchmarkStrictNewVersionPre(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	benchStrictNewVersion("1.0.0-alpha", b)
}

func BenchmarkNewVersionMeta(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	benchNewVersion("1.0.0+metadata", b)
}

func BenchmarkStrictNewVersionMeta(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	benchStrictNewVersion("1.0.0+metadata", b)
}

func BenchmarkNewVersionMetaDash(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	benchNewVersion("1.0.0-alpha.1+meta.data", b)
}

func BenchmarkStrictNewVersionMetaDash(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	benchStrictNewVersion("1.0.0-alpha.1+meta.data", b)
}
