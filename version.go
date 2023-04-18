package semver

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

var (
	// ErrInvalidSemVer is returned a version is found to be invalid when
	// being parsed.
	ErrInvalidSemVer = errors.New("invalid Semantic Version")

	// ErrEmptyString is returned when an empty string is passed in for parsing.
	ErrEmptyString = errors.New("version string empty")

	// ErrInvalidCharacters is returned when invalid characters are found as
	// part of a version
	ErrInvalidCharacters = errors.New("invalid characters in version")

	// ErrSegmentStartsZero is returned when a version segment starts with 0.
	// This is invalid in SemVer.
	ErrSegmentStartsZero = errors.New("version segment starts with 0")

	// ErrInvalidMetadata is returned when the metadata is an invalid format
	ErrInvalidMetadata = errors.New("invalid Metadata string")

	// ErrInvalidPrerelease is returned when the pre-release is an invalid format
	ErrInvalidPrerelease = errors.New("invalid Prerelease string")
)

// Version represents a single semantic version.
type Version struct {
	parts    []uint64
	pre      string
	metadata string
	original string
}

const num string = "0123456789"
const allowed string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-" + num

// StrictNewVersion parses a given version and returns an instance of Version or
// an error if unable to parse the version. Only parses valid semantic versions.
// Performs checking that can find errors within the version.
// If you want to allow optional v prefix, use the NewVersion() function.
func StrictNewVersion(v string) (*Version, error) {
	// Parsing here does not use RegEx in order to increase performance and reduce
	// allocations.

	if len(v) == 0 {
		return nil, ErrEmptyString
	}

	sv := &Version{
		original: v,
	}

	var partsString = v
	if i := strings.Index(v, "-"); i != -1 { // v1.2.3-some or v1.2.3-some+123
		partsString = v[:i] // [v1.2.3]
		sv.pre = v[i+1:]    // [some] or [some+123]
	} else { // v1.2.3 or v1.2.3+123
		if i := strings.Index(v, "+"); i != -1 { // v1.2.3+123
			partsString = v[:i]   // [v1.2.3]
			sv.metadata = v[i+1:] // [123]
		}
	}
	if sv.pre != "" { // v:(v1.2.3-some or v1.2.3-some+123) pre:(some or some+123)
		if i := strings.Index(sv.pre, "+"); i != -1 { // pre: some+123
			sv.metadata = sv.pre[i+1:]
			sv.pre = sv.pre[:i]
		}
	}

	// Split the parts
	parts := strings.Split(partsString, ".")
	if len(parts) == 0 {
		return nil, ErrInvalidSemVer
	}

	// Validate the number segments are valid. This includes only having positive
	// numbers and no leading 0's.
	sv.parts = make([]uint64, len(parts))
	var err error
	for i, p := range parts {
		sv.parts[i], err = strconv.ParseUint(parts[i], 10, 64)
		if err != nil {
			return nil, ErrInvalidCharacters
		}

		if len(p) > 1 && p[0] == '0' {
			return nil, ErrSegmentStartsZero
		}
	}

	// No prerelease or build metadata found so returning now as a fastpath.
	if sv.pre == "" && sv.metadata == "" {
		return sv, nil
	}

	if sv.pre != "" {
		if err = validatePrerelease(sv.pre); err != nil {
			return nil, err
		}
	}

	if sv.metadata != "" {
		if err = validateMetadata(sv.metadata); err != nil {
			return nil, err
		}
	}

	return sv, nil
}

// NewVersion parses a given version and returns an instance of Version or
// an error if unable to parse the version.
// This version allow a `v` prefix
func NewVersion(v string) (sv *Version, err error) {
	sv, err = StrictNewVersion(strings.TrimPrefix(v, "v"))
	if err != nil {
		return
	}

	sv.original = v

	return
}

func NewVersionByParts(nums ...uint64) *Version {
	return &Version{
		parts:    nums,
		original: joinNumbers(nums),
	}
}

func joinNumbers(nums []uint64) string {
	b := strings.Builder{}
	for i, n := range nums {
		if i > 0 {
			b.WriteRune('.')
		}
		b.WriteString(strconv.FormatUint(n, 10))
	}
	return b.String()
}

// MustParse parses a given version and panics on error.
func MustParse(v string) *Version {
	sv, err := NewVersion(v)
	if err != nil {
		panic(err)
	}
	return sv
}

func (v *Version) Copy() Version {
	return Version{
		parts:    append([]uint64{}, v.parts...),
		pre:      v.pre,
		metadata: v.metadata,
		original: v.original,
	}
}

// String converts a Version object to a string.
// Note, if the original version contained a leading v this version will not.
// See the Original() method to retrieve the original value. Semantic Versions
// don't contain a leading v per the spec. Instead it's optional on
// implementation.
func (v *Version) String() string {
	return strings.TrimPrefix(v.original, "v")
}

func (v *Version) updateOriginal() {
	buf := bytes.NewBufferString(v.originalVPrefix())
	buf.Grow(len(v.original) * 2)

	switch len(v.parts) {
	case 0:
	case 1:
		buf.WriteString(strconv.FormatUint(v.parts[0], 10))
	default:
		buf.WriteString(strconv.FormatUint(v.parts[0], 10))
		for _, s := range v.parts[1:] {
			buf.WriteByte('.')
			buf.WriteString(strconv.FormatUint(s, 10))
		}
	}

	if v.pre != "" {
		buf.WriteByte('-')
		buf.WriteString(v.pre)
	}

	if v.metadata != "" {
		buf.WriteByte('+')
		buf.WriteString(v.metadata)
	}

	v.original = buf.String()
}

// Original returns the original value passed in to be parsed.
func (v *Version) Original() string {
	return v.original
}

func (v *Version) PartsNumber() int {
	return len(v.parts)
}

func (v *Version) Part(part int) uint64 {
	if len(v.parts) >= part {
		return v.parts[part-1]
	}

	return 0
}

// Major returns the major version.
func (v *Version) Major() uint64 {
	return v.parts[0]
}

// Minor returns the minor version.
func (v *Version) Minor() uint64 {
	return v.Part(2)
}

// Patch returns the patch version.
func (v *Version) Patch() uint64 {
	return v.Part(3)
}

// Prerelease returns the pre-release version.
func (v *Version) Prerelease() string {
	return v.pre
}

// Metadata returns the metadata on the version.
func (v *Version) Metadata() string {
	return v.metadata
}

func (v *Version) isZero() bool {
	if v == nil {
		return true
	}

	for _, p := range v.parts {
		if p != 0 {
			return false
		}
	}

	return true
}

// originalVPrefix returns the original 'v' prefix if any.
func (v *Version) originalVPrefix() string {
	// Note, only lowercase v is supported as a prefix by the parser.
	if v.original != "" && v.original[:1] == "v" {
		return v.original[:1]
	}
	return ""
}

func (v *Version) ensurePartsNumber(expect int) {
	if len(v.parts) >= expect {
		return
	}

	v.parts = append(v.parts, make([]uint64, expect-len(v.parts))...)
}

// IncPart produces the next version on specific part
// If the part is not exist
// - increase the part number to specific
// - do same as last part
// If the part is last part
// - if the pre exist, will remove pre and metadata and won't increase the part
// - if the pre not exist, will remove metadata and increase the part
// If the part is not last part
// - will remove pre and metadata and increase the part
// - following parts will be set to zero
func (v *Version) IncPart(part int) Version {
	vNext := v.Copy()
	vNext.ensurePartsNumber(part)

	if len(vNext.parts) <= part { // last part
		if vNext.pre != "" {
			vNext.metadata = ""
			vNext.pre = ""
		} else {
			vNext.metadata = ""
			vNext.pre = ""
			vNext.parts[len(vNext.parts)-1]++
		}
	} else {
		vNext.metadata = ""
		vNext.pre = ""

		vNext.parts[part-1]++
		for i := part; i < len(vNext.parts); i++ {
			vNext.parts[i] = 0
		}
	}

	vNext.updateOriginal()

	return vNext
}

// IncPatch produces the next patch (3rd part) version.
// Same as IncPart(3)
func (v *Version) IncPatch() Version {
	return v.IncPart(3)
}

// IncMinor produces the next minor (2nd part) version.
// Same as IncPart(2)
func (v *Version) IncMinor() Version {
	return v.IncPart(2)
}

// IncMajor produces the next major (1st part) version.
// Same as IncPart(1)
func (v *Version) IncMajor() Version {
	return v.IncPart(1)
}

// IncLast produces the next version on last part
// Same as IncPart(len(v.PartsNumber()))
func (v *Version) IncLast() Version {
	return v.IncPart(len(v.parts))
}

// SetPrerelease defines the prerelease value.
// Value must not include the required 'hyphen' prefix.
func (v *Version) SetPrerelease(prerelease string) (Version, error) {
	vNext := v.Copy()
	if len(prerelease) > 0 {
		if err := validatePrerelease(prerelease); err != nil {
			return vNext, err
		}
	}
	vNext.pre = prerelease
	vNext.updateOriginal()
	return vNext, nil
}

// SetMetadata defines metadata value.
// Value must not include the required 'plus' prefix.
func (v *Version) SetMetadata(metadata string) (Version, error) {
	vNext := v.Copy()
	if len(metadata) > 0 {
		if err := validateMetadata(metadata); err != nil {
			return vNext, err
		}
	}
	vNext.metadata = metadata
	vNext.updateOriginal()
	return vNext, nil
}

// LessThan tests if one version is less than another one.
func (v *Version) LessThan(o *Version) bool {
	return v.Compare(o) < 0
}

// GreaterThan tests if one version is greater than another one.
func (v *Version) GreaterThan(o *Version) bool {
	return v.Compare(o) > 0
}

// Equal tests if two versions are equal to each other.
// Note, versions can be equal with different metadata since metadata
// is not considered part of the comparable version.
func (v *Version) Equal(o *Version) bool {
	return v.Compare(o) == 0
}

func maxPartsNumberOf(v1, v2 *Version) int {
	n1 := v1.PartsNumber()
	n2 := v2.PartsNumber()

	n := n1
	if n2 > n1 {
		n = n2
	}

	return n
}

// Compare compares this version to another one. It returns -1, 0, or 1 if
// the version smaller, equal, or larger than the other version.
//
// Versions are compared by X.Y.Z. Build metadata is ignored. Prerelease is
// lower than the version without a prerelease. Compare always takes into account
// prereleases. If you want to work with ranges using typical range syntaxes that
// skip prereleases if the range is not looking for them use constraints.
func (v *Version) Compare(o *Version) int {
	n := maxPartsNumberOf(v, o)

	// Compare parts from left to right
	//  If a difference is found return the comparison.
	for i := 1; i <= n; i++ {
		if d := compareSegment(v.Part(i), o.Part(i)); d != 0 {
			return d
		}
	}

	// At this point the version number parts are the same.
	ps := v.pre
	po := o.pre

	if ps == "" && po == "" {
		return 0
	}
	if ps == "" {
		return 1
	}
	if po == "" {
		return -1
	}

	return comparePrerelease(ps, po)
}

func (v *Version) updateBy(o *Version) {
	v.parts = o.parts
	v.pre = o.pre
	v.metadata = o.metadata
	v.original = o.original
}

// UnmarshalJSON implements JSON.Unmarshaler interface.
func (v *Version) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	temp, err := NewVersion(s)
	if err != nil {
		return err
	}
	v.updateBy(temp)

	return nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (v *Version) UnmarshalText(text []byte) error {
	temp, err := NewVersion(string(text))
	if err != nil {
		return err
	}

	*v = *temp

	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (v Version) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// MarshalJSON implements json.Marshaler interface.
func (v Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

// Scan implements the SQL.Scanner interface.
func (v *Version) Scan(value interface{}) error {
	var s string
	s, _ = value.(string)
	temp, err := NewVersion(s)
	if err != nil {
		return err
	}
	v.updateBy(temp)
	return nil
}

// Value implements the Driver.Valuer interface.
func (v Version) Value() (driver.Value, error) {
	return v.String(), nil
}

func compareSegment(v, o uint64) int {
	if v < o {
		return -1
	}
	if v > o {
		return 1
	}

	return 0
}

func comparePrerelease(v, o string) int {
	// split the prelease versions by their part. The separator, per the spec,is a `.`
	sparts := strings.Split(v, ".")
	oparts := strings.Split(o, ".")

	// Find the longer length of the parts to know how many loop iterations to
	// go through.
	slen := len(sparts)
	olen := len(oparts)

	l := slen
	if olen > slen {
		l = olen
	}

	// Iterate over each part of the prereleases to compare the differences.
	for i := 0; i < l; i++ {
		// Since the lentgh of the parts can be different we need to create
		// a placeholder. This is to avoid out of bounds issues.
		stemp := ""
		if i < slen {
			stemp = sparts[i]
		}

		otemp := ""
		if i < olen {
			otemp = oparts[i]
		}

		d := comparePrePart(stemp, otemp)
		if d != 0 {
			return d
		}
	}

	// Reaching here means two versions are of equal value but have different
	// metadata (the part following a +). They are not identical in string form
	// but the version comparison finds them to be equal.
	return 0
}

func comparePrePart(s, o string) int {
	// Fastpath if they are equal
	if s == o {
		return 0
	}

	// When s or o are empty we can use the other in an attempt to determine
	// the response.
	if s == "" {
		if o != "" {
			return -1
		}
		return 1
	}

	if o == "" {
		if s != "" {
			return 1
		}
		return -1
	}

	// When comparing strings "99" is greater than "103". To handle
	// cases like this we need to detect numbers and compare them. According
	// to the semver spec, numbers are always positive. If there is a - at the
	// start like -99 this is to be evaluated as an alphanum. numbers always
	// have precedence over alphanum. Parsing as Uints because negative numbers
	// are ignored.

	oi, n1 := strconv.ParseUint(o, 10, 64)
	si, n2 := strconv.ParseUint(s, 10, 64)

	// The case where both are strings compare the strings
	if n1 != nil && n2 != nil {
		if s > o {
			return 1
		}
		return -1
	} else if n1 != nil {
		// o is a string and s is a number
		return -1
	} else if n2 != nil {
		// s is a string and o is a number
		return 1
	}
	// Both are numbers
	if si > oi {
		return 1
	}
	return -1

}

// Like strings.ContainsAny but does an only instead of any.
func containsOnly(s string, comp string) bool {
	return strings.IndexFunc(s, func(r rune) bool {
		return !strings.ContainsRune(comp, r)
	}) == -1
}

// From the spec, "Identifiers MUST comprise only
// ASCII alphanumerics and hyphen [0-9A-Za-z-]. Identifiers MUST NOT be empty.
// Numeric identifiers MUST NOT include leading zeroes.". These segments can
// be dot separated.
func validatePrerelease(p string) error {
	eparts := strings.Split(p, ".")
	for _, p := range eparts {
		if containsOnly(p, num) {
			if len(p) > 1 && p[0] == '0' {
				return ErrSegmentStartsZero
			}
		} else if !containsOnly(p, allowed) {
			return ErrInvalidPrerelease
		}
	}

	return nil
}

// From the spec, "Build metadata MAY be denoted by
// appending a plus sign and a series of dot separated identifiers immediately
// following the patch or pre-release version. Identifiers MUST comprise only
// ASCII alphanumerics and hyphen [0-9A-Za-z-]. Identifiers MUST NOT be empty."
func validateMetadata(m string) error {
	eparts := strings.Split(m, ".")
	for _, p := range eparts {
		if !containsOnly(p, allowed) {
			return ErrInvalidMetadata
		}
	}
	return nil
}
