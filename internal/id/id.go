package id

import "strings"

var idCache = map[string]string{}

func IsValidAlias(alias string) bool {
	if alias == "" {
		return false
	}

	for _, r := range alias {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' {
			continue
		}

		return false
	}

	return true
}

func IsValidProjectId(id string) bool {
	if id == "" {
		return false
	}

	last := len(id) - 1

	startsWithAt := false
	containsSlash := false

	for i, r := range id {
		if i == 0 {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '@' {
				if r == '@' {
					startsWithAt = true
				}
				continue
			} else {
				return false
			}
		}

		if i == last {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				continue
			} else {
				return false
			}
		}

		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == '/' {
			if r == '/' {
				containsSlash = true
			}
			continue
		} else {
			return false
		}
	}

	if startsWithAt && !containsSlash {
		return false
	}

	return true
}

func Slugify(id string) string {
	builder := []rune{}
	for _, r := range id {
		switch r {
		case '-', '_', ' ', '.', '/', ':':
			builder = append(builder, '-')
		default:
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				builder = append(builder, r)
			}
		}
	}

	return string(builder)
}

func Sanitize(input string) string {
	var sb strings.Builder
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}

	s := sb.String()
	s = strings.Trim(s, "-")

	var collapsed strings.Builder
	lastHyphen := false
	for _, r := range s {
		if r == '-' {
			if !lastHyphen {
				collapsed.WriteRune(r)
				lastHyphen = true
			}
		} else {
			collapsed.WriteRune(r)
			lastHyphen = false
		}
	}

	return collapsed.String()
}

func Convert(name string) string {
	if id, exists := idCache[name]; exists {
		return id
	}

	builder := []rune{}
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}

		if i > 0 && r == '-' {
			last := builder[len(builder)-1]
			if last == '-' {
				continue
			}
		}

		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder = append(builder, r)
		} else if r == '-' || r == '_' || r == '.' || r == '/' || r == ':' || r == ' ' {
			builder = append(builder, r)
		} else {
			continue
		}
	}

	id := string(builder)
	idCache[name] = id
	return id
}

func IsValidId(id string) bool {
	if id == "" {
		return false
	}

	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		} else {
			return false
		}
	}
	return true
}

func IsValidName(name string) bool {
	if name == "" {
		return false
	}

	for _, r := range name {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == ' ' || r == ':' || r == '/' || r == '.' {
			continue
		} else {
			return false
		}
	}
	return true
}
