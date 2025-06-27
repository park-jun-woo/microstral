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
func (m *Metadata) HasRoles(roles []uint32) bool {

	return false
}

// Role 추가
func (m *Metadata) Role(role uint32) *Metadata {
	m.Roles = append(m.Roles, role)
	return m
}

// Param 추가
func (m *Metadata) Param(name string, def string, typ uint32) *Metadata {
	m.Params[name] = param.Param{
		Name:    name,
		Default: def,
		Type:    typ,
	}
	return m
}

// Flag Param 추가
func (m *Metadata) Flag(name string, def string, flag uint64) *Metadata {
	m.Params[name] = param.Param{
		Name:    name,
		Default: def,
		Type:    param.FLAG,
		Flag:    flag,
	}
	return m
}

// Regexp Param 추가
func (m *Metadata) Regexp(name string, def string, pattern string) *Metadata {
	m.Params[name] = param.Param{
		Name:    name,
		Default: def,
		Type:    param.REGEX,
		Regex:   regexp.MustCompile(pattern),
	}
	return m
}
