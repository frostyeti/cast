package id

var idCache = map[string]string{}

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
