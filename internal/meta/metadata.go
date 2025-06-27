package meta

import (
	"regexp"

	"parkjunwoo.com/microstral/pkg/param"
)

// MetadataMap : 메타데이터 맵 string -> *Metadata
type MetadataMap map[string]*Metadata

// Metadata : 메타데이터 구조체
type Metadata struct {
	Roles  []uint32
	Params map[string]param.Param
}

// Role이 있는지 확인
func (c *Metadata) HasRoles(roles []uint32) bool {

	return false
}

// Role 추가
func (c *Metadata) Role(role uint32) *Metadata {
	c.Roles = append(c.Roles, role)
	return c
}

// Param 추가
func (c *Metadata) Param(name string, def string, typ uint32) *Metadata {
	c.Params[name] = param.Param{
		Name:    name,
		Default: def,
		Type:    typ,
	}
	return c
}

// Flag Param 추가
func (c *Metadata) Flag(name string, def string, flag uint64) *Metadata {
	c.Params[name] = param.Param{
		Name:    name,
		Default: def,
		Type:    param.FLAG,
		Flag:    flag,
	}
	return c
}

// Regexp Param 추가
func (c *Metadata) Regexp(name string, def string, pattern string) *Metadata {
	c.Params[name] = param.Param{
		Name:    name,
		Default: def,
		Type:    param.REGEX,
		Regex:   regexp.MustCompile(pattern),
	}
	return c
}
