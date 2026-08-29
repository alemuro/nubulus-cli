package client

import (
	"regexp"
	"strings"
)

var labelRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
var recordLabelRe = regexp.MustCompile(`^_?[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

const maxLabelLen = 63

// NormalizeZoneName lowercases and trims a zone name and checks validity.
func NormalizeZoneName(raw string) (string, bool) {
	name := strings.ToLower(strings.TrimSpace(raw))
	name = strings.TrimSuffix(name, ".")
	if name == "" || len(name) > 253 {
		return "", false
	}
	if strings.ContainsAny(name, "/:@ ") {
		return "", false
	}
	labels := strings.Split(name, ".")
	if len(labels) < 2 {
		return "", false
	}
	for _, l := range labels {
		if len(l) > maxLabelLen || !labelRe.MatchString(l) {
			return "", false
		}
	}
	return name, true
}

func validRecordName(name string) bool {
	trimmed := strings.TrimSuffix(name, ".")
	if trimmed == "" || len(trimmed) > 253 {
		return false
	}
	if strings.ContainsAny(trimmed, "/:@ ") {
		return false
	}
	labels := strings.Split(trimmed, ".")
	if len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if len(l) > maxLabelLen || !recordLabelRe.MatchString(l) {
			return false
		}
	}
	return true
}

// NormalizeOwnerName lowercases an owner name and appends trailing dot.
func NormalizeOwnerName(raw string) string {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return ""
	}
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	return name
}

// QualifyName resolves the name of a record against its zone.
func QualifyName(name, zone string) (string, bool) {
	zoneName, ok := NormalizeZoneName(zone)
	if !ok {
		return "", false
	}
	apex := zoneName + "."

	raw := strings.ToLower(strings.TrimSpace(name))
	if raw == "" {
		return "", false
	}
	if raw == "@" {
		return apex, true
	}

	qualified := raw
	if !strings.HasSuffix(qualified, ".") {
		if qualified == zoneName || strings.HasSuffix(qualified, "."+zoneName) {
			qualified += "."
		} else {
			qualified = qualified + "." + apex
		}
	}

	if qualified != apex && !strings.HasSuffix(qualified, "."+apex) {
		return "", false
	}
	if len(qualified) > 254 {
		return "", false
	}

	check := strings.TrimPrefix(qualified, "*.")
	if !validRecordName(check) {
		return "", false
	}

	return qualified, true
}

// IsApex returns true if qualifiedName is the zone apex.
func IsApex(qualifiedName, zone string) bool {
	zoneName, ok := NormalizeZoneName(zone)
	if !ok {
		return false
	}
	return NormalizeOwnerName(qualifiedName) == zoneName+"."
}

// ManagedAtApex are the record sets the platform manages at the apex.
var ManagedAtApex = map[string]struct{}{
	"SOA": {},
	"NS":  {},
}

// ForbiddenTypes are types prohibited by the Nubulus DNS API.
var ForbiddenTypes = map[string]struct{}{
	"DNSKEY": {}, "RRSIG": {}, "NSEC": {}, "NSEC3": {}, "NSEC3PARAM": {},
	"CDS": {}, "CDNSKEY": {}, "DNAME": {},
	"AXFR": {}, "IXFR": {}, "ANY": {}, "OPT": {}, "TSIG": {}, "TKEY": {},
}

const (
	MinTTL         = 60
	MaxTTL         = 604800
	MaxRRsetValues = 100
)
