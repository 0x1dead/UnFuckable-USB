package main

import (
	"os"
	"path/filepath"
	"strings"
)

type ExclusionRule struct {
	Pattern  string
	Type     string // "exact", "prefix", "suffix", "contains", "glob"
	IsDir    bool
	Comment  string
}

// ParseExclusions parses exclusion file
func ParseExclusions(content string) []ExclusionRule {
	var rules []ExclusionRule

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty and comments
		if line == "" || line[0] == '#' {
			continue
		}

		rule := ExclusionRule{Pattern: line}

		// Detect type
		if strings.HasPrefix(line, "*") && strings.HasSuffix(line, "*") {
			rule.Type = "contains"
			rule.Pattern = line[1 : len(line)-1]
		} else if strings.HasPrefix(line, "*") {
			rule.Type = "suffix"
			rule.Pattern = line[1:]
		} else if strings.HasSuffix(line, "*") {
			rule.Type = "prefix"
			rule.Pattern = line[:len(line)-1]
		} else if strings.HasSuffix(line, "/") {
			rule.Type = "prefix"
			rule.IsDir = true
			rule.Pattern = line[:len(line)-1]
		} else {
			rule.Type = "exact"
		}

		rules = append(rules, rule)
	}

	return rules
}

// MatchRule checks if path matches rule
func MatchRule(path string, rule ExclusionRule) bool {
	// Normalize separators
	path = filepath.ToSlash(path)
	pattern := filepath.ToSlash(rule.Pattern)

	switch rule.Type {
	case "exact":
		return path == pattern
	case "prefix":
		return strings.HasPrefix(path, pattern)
	case "suffix":
		return strings.HasSuffix(path, pattern)
	case "contains":
		return strings.Contains(path, pattern)
	}

	return false
}

// MatchAny checks if path matches any rule
func MatchAny(path string, rules []ExclusionRule) bool {
	for _, rule := range rules {
		if MatchRule(path, rule) {
			return true
		}
	}
	return false
}

// AddExclusion adds exclusion to config
func AddExclusion(pattern string) {
	for _, e := range AppConfig.Exclusions {
		if e == pattern {
			return
		}
	}
	AppConfig.Exclusions = append(AppConfig.Exclusions, pattern)
	SaveConfig()
}

// RemoveExclusion removes exclusion from config
func RemoveExclusion(pattern string) {
	var newExcl []string
	for _, e := range AppConfig.Exclusions {
		if e != pattern {
			newExcl = append(newExcl, e)
		}
	}
	AppConfig.Exclusions = newExcl
	SaveConfig()
}

// GetExclusions returns all exclusions
func GetExclusions() []string {
	return AppConfig.Exclusions
}

// CreateExcludeFile creates .unfuckable.exclude on drive
func CreateExcludeFile(drivePath string, patterns []string) error {
	content := "# UnFuckable USB Exclusions\n"
	content += "# One pattern per line\n"
	content += "# Examples:\n"
	content += "#   portable/*     - exclude portable folder\n"
	content += "#   *.exe          - exclude all exe files\n"
	content += "#   *secret*       - exclude anything with 'secret'\n"
	content += "#   backup/        - exclude backup directory\n"
	content += "\n"

	for _, p := range patterns {
		content += p + "\n"
	}

	return os.WriteFile(filepath.Join(drivePath, ExcludeFile), []byte(content), 0644)
}

// DefaultExclusions returns default exclusion patterns
func DefaultExclusions() []string {
	return []string{
		"unfuckable-usb*",
		"UnFuckable*",
		".unfuckable*",
		"portable/*",
		"Portable/*",
		"System Volume Information/*",
		"$RECYCLE.BIN/*",
		".Trash*",
		".DS_Store",
		"Thumbs.db",
		"desktop.ini",
	}
}
