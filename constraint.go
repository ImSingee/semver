package semver

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Constraints is one or more constraint that a semantic version can be
// checked against.
type Constraints struct {
	constraints [][]*constraint
}

// NewConstraint returns a Constraints instance that a Version instance can
// be checked against. If there is a parse error it will be returned.
func NewConstraint(c string) (*Constraints, error) {
	// Rewrite - ranges into a comparison operation.
	c = rewriteRange(c)

	ors := strings.Split(c, "||")
	or := make([][]*constraint, len(ors))
	for k, v := range ors {

		// TODO: Find a way to validate and fetch all the constraints in a simpler form

		// Validate the segment
		if !validConstraintRegex.MatchString(v) {
			return nil, fmt.Errorf("improper constraint: %s", v)
		}

		cs := findConstraintRegex.FindAllString(v, -1)
		if cs == nil {
			cs = append(cs, v)
		}
		result := make([]*constraint, len(cs))
		for i, s := range cs {
			pc, err := parseConstraint(s)
			if err != nil {
				return nil, err
			}

			result[i] = pc
		}
		or[k] = result
	}

	o := &Constraints{constraints: or}
	return o, nil
}

// Check tests if a version satisfies the constraints.
func (cs Constraints) Check(v *Version) bool {
	// TODO(mattfarina): For v4 of this library consolidate the Check and Validate
	// functions as the underlying functions make that possible now.
	// loop over the ORs and check the inner ANDs
	for _, o := range cs.constraints {
		joy := true
		for _, c := range o {
			if check, _ := c.check(v); !check {
				joy = false
				break
			}
		}

		if joy {
			return true
		}
	}

	return false
}

// Validate checks if a version satisfies a constraint. If not a slice of
// reasons for the failure are returned in addition to a bool.
func (cs Constraints) Validate(v *Version) (bool, []error) {
	// loop over the ORs and check the inner ANDs
	var e []error

	// Capture the prerelease message only once. When it happens the first time
	// this var is marked
	var prerelesase bool
	for _, o := range cs.constraints {
		joy := true
		for _, c := range o {
			// Before running the check handle the case there the version is
			// a prerelease and the check is not searching for prereleases.
			if c.con.pre == "" && v.pre != "" {
				if !prerelesase {
					em := fmt.Errorf("%s is a prerelease version and the constraint is only looking for release versions", v)
					e = append(e, em)
					prerelesase = true
				}
				joy = false
			} else {
				if _, err := c.check(v); err != nil {
					e = append(e, err)
					joy = false
				}
			}
		}

		if joy {
			return true, []error{}
		}
	}

	return false, e
}

func (cs Constraints) String() string {
	buf := make([]string, len(cs.constraints))
	var tmp bytes.Buffer

	for k, v := range cs.constraints {
		tmp.Reset()
		vlen := len(v)
		for kk, c := range v {
			tmp.WriteString(c.string())

			// Space separate the AND conditions
			if vlen > 1 && kk < vlen-1 {
				tmp.WriteString(" ")
			}
		}
		buf[k] = tmp.String()
	}

	return strings.Join(buf, " || ")
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (cs *Constraints) UnmarshalText(text []byte) error {
	temp, err := NewConstraint(string(text))
	if err != nil {
		return err
	}

	*cs = *temp

	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (cs Constraints) MarshalText() ([]byte, error) {
	return []byte(cs.String()), nil
}

var constraintOps map[string]cfunc
var constraintRegex *regexp.Regexp
var constraintRangeRegex *regexp.Regexp

// Used to find individual constraints within a multi-constraint string
var findConstraintRegex *regexp.Regexp

// Used to validate an segment of ANDs is valid
var validConstraintRegex *regexp.Regexp

const cvRegex string = `v?([0-9|x|X|\*]+)((\.[0-9|x|X|\*]+)*)` +
	`(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?` +
	`(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?`

func init() {
	constraintOps = map[string]cfunc{
		"":   constraintTilde,
		"=":  constraintEqual,
		"!=": constraintNotEqual,
		">":  constraintGreaterThan,
		"<":  constraintLessThan,
		">=": constraintGreaterThanEqual,
		"=>": constraintGreaterThanEqual,
		"<=": constraintLessThanEqual,
		"=<": constraintLessThanEqual,
		"~":  constraintTilde,
		"~>": constraintTilde,
		"^":  constraintCaret,
	}

	ops := `=||!=|>|<|>=|=>|<=|=<|~|~>|\^`

	constraintRegex = regexp.MustCompile(fmt.Sprintf(
		`^\s*(%s)\s*(%s)\s*$`,
		ops,
		cvRegex))

	constraintRangeRegex = regexp.MustCompile(fmt.Sprintf(
		`\s*(%s)\s+-\s+(%s)\s*`,
		cvRegex, cvRegex))

	findConstraintRegex = regexp.MustCompile(fmt.Sprintf(
		`(%s)\s*(%s)`,
		ops,
		cvRegex))

	// The first time a constraint shows up will look slightly different from
	// future times it shows up due to a leading space or comma in a given
	// string.
	validConstraintRegex = regexp.MustCompile(fmt.Sprintf(
		`^(\s*(%s)\s*(%s)\s*)((?:\s+|,\s*)(%s)\s*(%s)\s*)*$`,
		ops,
		cvRegex,
		ops,
		cvRegex))
}

// An individual constraint
type constraint struct {
	// The version used in the constraint check. For example, if a constraint
	// is '<= 2.0.0' the con a version instance representing 2.0.0.
	con *Version

	// The original parsed version (e.g., 4.x from != 4.x)
	orig string

	// The original operator for the constraint
	origfunc string

	// When an x is used as part of the version (e.g., 1.x)
	dirtyPart int
}

// Check if a version meets the constraint
func (c *constraint) check(v *Version) (bool, error) {
	return constraintOps[c.origfunc](v, c)
}

// String prints an individual constraint into a string
func (c *constraint) string() string {
	return c.origfunc + c.orig
}

type cfunc func(v *Version, c *constraint) (bool, error)

func parseConstraint(c string) (*constraint, error) {
	if len(c) > 0 {
		m := constraintRegex.FindStringSubmatch(c)
		if m == nil {
			return nil, fmt.Errorf("improper constraint: %s", c)
		}

		cs := &constraint{
			orig:     m[2],
			origfunc: m[1],
		}

		ver := m[2]

		var verWithoutTrailing string
		var preAndMeta string
		i := strings.IndexAny(ver, "+-")
		if i == -1 {
			verWithoutTrailing = ver
		} else {
			verWithoutTrailing = ver[:i]
			preAndMeta = ver[i:]
		}

		dirtyPart := 0
		verParts := strings.Split(verWithoutTrailing, ".")
		for i, p := range verParts {
			if isX(p) {
				dirtyPart = i + 1
				break
			}
		}

		if dirtyPart > 0 {
			if dirtyPart != len(verParts) { // dirty part must also be last part
				for i := dirtyPart; i < len(verParts); i++ {
					if !isX(verParts[i]) {
						return nil, fmt.Errorf("improper constraint: %s", c)
					}
				}
			}

			if dirtyPart == 1 {
				verWithoutTrailing = "0" // remove first x
			} else {
				verWithoutTrailing = strings.Join(verParts[:dirtyPart-1], ".") // trim .x
			}
		}

		con, err := NewVersion(verWithoutTrailing + preAndMeta)
		if err != nil {
			// The constraintRegex should catch any regex parsing errors. So,
			// we should never get here.
			return nil, errors.New("constraint Parser Error")
		}

		cs.con = con
		cs.dirtyPart = dirtyPart

		return cs, nil
	}

	// The rest is the special case where an empty string was passed in which
	// is equivalent to * or >=0.0.0

	cs := &constraint{
		con: &Version{
			parts:    []uint64{0},
			original: "0",
		},
		orig:      c,
		origfunc:  "",
		dirtyPart: 1,
	}
	return cs, nil
}

func maxNonDirtyPartsNumberOf(v *Version, c *constraint) int {
	n := maxPartsNumberOf(v, c.con)

	if c.dirtyPart > 0 && n > c.dirtyPart {
		return c.dirtyPart
	}

	return n
}

// Constraint functions
func constraintNotEqual(v *Version, c *constraint) (bool, error) {
	if c.dirtyPart > 0 {
		// If there is a pre-release on the version but the constraint isn't looking
		// for them assume that pre-releases are not compatible. See issue 21 for
		// more details.
		if v.Prerelease() != "" && c.con.Prerelease() == "" {
			return false, fmt.Errorf("%s is a prerelease version and the constraint is only looking for release versions", v)
		}

		for i := 1; i < maxNonDirtyPartsNumberOf(v, c); i++ {
			if c.con.Part(i) != v.Part(i) {
				return true, nil
			}
		}

		// dirty part is reach, and still equal

		return false, fmt.Errorf("%s is equal to %s", v, c.orig)
	}

	eq := v.Equal(c.con)
	if eq {
		return false, fmt.Errorf("%s is equal to %s", v, c.orig)
	}

	return true, nil
}

func constraintGreaterThan(v *Version, c *constraint) (bool, error) {
	// If there is a pre-release on the version but the constraint isn't looking
	// for them assume that pre-releases are not compatible. See issue 21 for
	// more details.
	if v.Prerelease() != "" && c.con.Prerelease() == "" {
		return false, fmt.Errorf("%s is a prerelease version and the constraint is only looking for release versions", v)
	}
	if c.dirtyPart > 0 {
		return false, fmt.Errorf("%s contains > operator which is not supported for .x", c.orig)
	}

	if v.Compare(c.con) == 1 {
		return true, nil
	}
	return false, fmt.Errorf("%s is less than or equal to %s", v, c.orig)
}

func constraintLessThan(v *Version, c *constraint) (bool, error) {
	// If there is a pre-release on the version but the constraint isn't looking
	// for them assume that pre-releases are not compatible. See issue 21 for
	// more details.
	if v.Prerelease() != "" && c.con.Prerelease() == "" {
		return false, fmt.Errorf("%s is a prerelease version and the constraint is only looking for release versions", v)
	}
	if c.dirtyPart > 0 {
		return false, fmt.Errorf("%s contains < operator which is not supported for .x", c.orig)
	}

	if v.Compare(c.con) == -1 {
		return true, nil
	}
	return false, fmt.Errorf("%s is greater than or equal to %s", v, c.orig)
}

func constraintGreaterThanEqual(v *Version, c *constraint) (bool, error) {
	// If there is a pre-release on the version but the constraint isn't looking
	// for them assume that pre-releases are not compatible. See issue 21 for
	// more details.
	if v.Prerelease() != "" && c.con.Prerelease() == "" {
		return false, fmt.Errorf("%s is a prerelease version and the constraint is only looking for release versions", v)
	}
	if c.dirtyPart > 0 {
		return false, fmt.Errorf("%s contains >= operator which is not supported for .x", c.orig)
	}

	eq := v.Compare(c.con) >= 0
	if eq {
		return true, nil
	}
	return false, fmt.Errorf("%s is less than %s", v, c.orig)
}

func constraintLessThanEqual(v *Version, c *constraint) (bool, error) {
	// If there is a pre-release on the version but the constraint isn't looking
	// for them assume that pre-releases are not compatible. See issue 21 for
	// more details.
	if v.Prerelease() != "" && c.con.Prerelease() == "" {
		return false, fmt.Errorf("%s is a prerelease version and the constraint is only looking for release versions", v)
	}
	if c.dirtyPart > 0 {
		return false, fmt.Errorf("%s contains <= operator which is not supported for .x", c.orig)
	}

	if v.Compare(c.con) <= 0 {
		return true, nil
	}

	return false, fmt.Errorf("%s is greater than %s", v, c.orig)
}

func (c *constraint) dirtyPartOrLen() int {
	if c.dirtyPart > 0 {
		return c.dirtyPart
	}

	return c.con.PartsNumber()
}

// ~*, ~>* --> >= 0.0.0 (any)
// ~2, ~2.x, ~2.x.x, ~>2, ~>2.x ~>2.x.x --> >=2.0.0, <3.0.0
// ~2.0, ~2.0.x, ~>2.0, ~>2.0.x --> >=2.0.0, <2.1.0
// ~1.2, ~1.2.x, ~>1.2, ~>1.2.x --> >=1.2.0, <1.3.0
// ~1.2.3, ~>1.2.3 --> >=1.2.3, <1.3.0
// ~1.2.0, ~>1.2.0 --> >=1.2.0, <1.3.0
func constraintTilde(v *Version, c *constraint) (bool, error) {
	if c.dirtyPart == 1 { // allow *
		return true, nil
	}
	if c.origfunc == "~" && c.con.isZero() { // ~0.0.0 is a special case where all constraints are accepted
		return true, nil
	}

	// If there is a pre-release on the version but the constraint isn't looking
	// for them assume that pre-releases are not compatible. See issue 21 for
	// more details.
	if v.Prerelease() != "" && c.con.Prerelease() == "" {
		return false, fmt.Errorf("%s is a prerelease version and the constraint is only looking for release versions", v)
	}

	if v.LessThan(c.con) {
		return false, fmt.Errorf("%s is less than %s", v, c.orig)
	}

	partNumber := c.con.PartsNumber()

	if v.Major() != c.con.Major() {
		return false, fmt.Errorf("%s does not have same major version as %s", v, c.orig)
	}

	if partNumber <= 1 {
		return true, nil
	}

	if v.Minor() != c.con.Minor() {
		return false, fmt.Errorf("%s does not have same major and minor version as %s", v, c.orig)
	}

	if partNumber <= 3 {
		return true, nil
	}

	for i := 3; i < partNumber; i++ {
		if v.Part(i) != c.con.Part(i) {
			return false, fmt.Errorf("%s does not have same %d part version as %s", v, i, c.orig)
		}
	}

	return true, nil
}

func constraintEqual(v *Version, c *constraint) (bool, error) {
	eq := v.Equal(c.con)
	if eq {
		return true, nil
	}

	return false, fmt.Errorf("%s is not equal to %s", v, c.orig)
}

func (v *Version) leftZeroPartNumber() int {
	zeroes := 0

	for _, p := range v.parts {
		if p == 0 {
			zeroes++
		} else {
			break
		}
	}

	return zeroes
}

// ^*      -->  (any)
// ^1.2.3  -->  >=1.2.3 <2.0.0
// ^1.2    -->  >=1.2.0 <2.0.0
// ^1      -->  >=1.0.0 <2.0.0
// ^0.2.3  -->  >=0.2.3 <0.3.0
// ^0.2    -->  >=0.2.0 <0.3.0
// ^0.0.3  -->  >=0.0.3 <0.0.4
// ^0.0    -->  >=0.0.0 <0.1.0
// ^0      -->  >=0.0.0 <1.0.0
func constraintCaret(v *Version, c *constraint) (bool, error) {
	// If there is a pre-release on the version but the constraint isn't looking
	// for them assume that pre-releases are not compatible. See issue 21 for
	// more details.
	if v.Prerelease() != "" && c.con.Prerelease() == "" {
		return false, fmt.Errorf("%s is a prerelease version and the constraint is only looking for release versions", v)
	}

	leftZeroPartNumber := c.con.leftZeroPartNumber()
	shouldEqualNumber := leftZeroPartNumber + 1
	if leftZeroPartNumber == c.con.PartsNumber() { // all zero
		shouldEqualNumber = leftZeroPartNumber
	}

	for i := 1; i <= shouldEqualNumber; i++ {
		if v.Part(i) != c.con.Part(i) {
			switch i {
			case 1:
				return false, fmt.Errorf("%s does not have same major version as %s", v, c.orig)
			case 2:
				return false, fmt.Errorf("%s does not have same minor version as %s", v, c.orig)
			default:
				return false, fmt.Errorf("%s does not have same %d part version as %s", v, i, c.orig)
			}
		}
	}

	// version must be greater or equal than constraint
	if v.LessThan(c.con) {
		return false, fmt.Errorf("%s is less than %s", v, c.orig)
	}

	return true, nil
}

func isX(x string) bool {
	switch x {
	case "x", "*", "X":
		return true
	default:
		return false
	}
}

func rewriteRange(i string) string {
	m := constraintRangeRegex.FindAllStringSubmatch(i, -1)
	if m == nil {
		return i
	}
	o := i
	for _, v := range m {
		t := fmt.Sprintf(">= %s, <= %s ", v[1], v[11])
		o = strings.Replace(o, v[0], t, 1)
	}

	return o
}
