# SemVer

[![Go Reference](https://pkg.go.dev/badge/github.com/ImSingee/semver.svg)](https://pkg.go.dev/github.com/ImSingee/semver) [![Test Status](https://github.com/ImSingee/semver/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/ImSingee/semver/actions/workflows/test.yml?query=branch%3Amaster) [![codecov](https://codecov.io/gh/ImSingee/semver/branch/master/graph/badge.svg?token=RWV4ZYS1DH)](https://codecov.io/gh/ImSingee/semver) [![Go Report Card](https://goreportcard.com/badge/github.com/ImSingee/semver)](https://goreportcard.com/report/github.com/ImSingee/semver)

https://github.com/Masterminds/semver, but support any numbers version part. 

## Install and Use

```bash
go get github.com/ImSingee/semver
```

See [documents](https://pkg.go.dev/github.com/ImSingee/semver)

## Version Compare & Constraint Rules

Help me translate into English, wrap it in a pre block in markdown format for me; this is a Github Readme

**Supported constraint syntax**

Basic comparison
- `=` or no symbol
- `!=`
- `>`
- `>=`
- `<`
- `<=`

Logical operations
- AND: Multiple matching conditions can be separated by `,`

Range
- `V1 - V2` is equivalent to `>= V1, <= V2`

Wildcard
- A single `*` matches any version number
- `1.2.x` is equivalent to `>=1.2, <1.3`

Minor version
- `~1.2.3.4` is equivalent to `>= 1.2.3.4, < 1.2.4`
- `~1.2.3` is equivalent to `>= 1.2.3, < 1.3`
- **Special** `~1.2` is equivalent to `>= 1.2, < 1.3`
- **Special** `~1` is equivalent to `>= 1, < 2`

Major version
- `^1.2.3` is equivalent to `>= 1.2.3, < 2`
- `^1.2` is equivalent to `>= 1.2, < 2`
- `^1` is equivalent to `>= 1, < 2`

**Matching different number of digits**

Compare by padding 0 to the maximum significant digit

According to this rule, there are `1.1` == `1.1.0` == `1.1.0.0`

**Matching with pre-release version numbers**

When excluding the same pre-release version numbers, the pre-release version number match will be performed

- One has a pre-release version number, the other does not: the one without is larger
    - `1.0-hello` < `1.0`
- Both have pre-release version numbers: perform pre-release version number ASCII comparison, which will be grouped by `.` and numeric groups will be matched numerically
    - `1.0-alpha` < `1.0-beta`
    - `1.0-alpha10` < `1.0-alpha2`
    - `1.0-alpha.2` < `1.0-alpha.10`
    - `1.0-alpha.100` < `1.0-beta`

**Metadata information**

Only when `=` is matched and the constraint contains metadata will the metadata be checked, otherwise the metadata will be ignored

- `=1.0.0+hello` can only match `1.0.0+hello` but not `1.0.0` (however, it can match `1.0+hello` `1+hello`)
- `=1.0.0` can match `1.0.0` `1.0.0+anything`
- `>1.0.0` can match `1.0.1` `1.0.1+anything`
- `>1.0.0+hello` is invalid
