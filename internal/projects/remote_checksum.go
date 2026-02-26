package projects

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

var projectChecksumCache = map[string]bool{} // Dir -> matches

func VerifyChecksumAndRefresh(p *Project) bool {
	if match, ok := projectChecksumCache[p.Dir]; ok {
		return match
	}

	content, err := os.ReadFile(p.File)
	if err != nil {
		projectChecksumCache[p.Dir] = false
		return false
	}

	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	castDir := p.CastDir
	if castDir == "" {
		castDir = filepath.Join(p.Dir, ".cast") // Fallback just in case
	}
	stateFile := filepath.Join(castDir, "state.json")

	// Get a stable relative name for this file based on CastDir's parent
	rootDir := filepath.Dir(castDir)
	relFile, err := filepath.Rel(rootDir, p.File)
	if err != nil || relFile == "" || relFile == "." {
		// fallback to absolute if Rel fails
		relFile = p.File
	}

	type State struct {
		Checksum string            `json:"checksum,omitempty"` // legacy
		Files    map[string]string `json:"files,omitempty"`
	}

	var state State
	state.Files = make(map[string]string)
	matches := false

	if stateData, err := os.ReadFile(stateFile); err == nil {
		if json.Unmarshal(stateData, &state) == nil {
			if state.Files != nil && len(state.Files) > 0 {
				if fileHash, exists := state.Files[relFile]; exists && fileHash == hashStr {
					matches = true
				}
			} else if state.Checksum == hashStr {
				// legacy support
				matches = true
				if state.Files == nil {
					state.Files = make(map[string]string)
				}
				state.Files[relFile] = state.Checksum
			}
		}
	}

	if state.Files == nil {
		state.Files = make(map[string]string)
	}

	if !matches {
		// Clear local cache for tasks if checksum doesn't match
		os.RemoveAll(filepath.Join(castDir, "tasks"))
		// Clear local cache for modules if checksum doesn't match
		os.RemoveAll(filepath.Join(castDir, "modules"))

		// Write new state
		os.MkdirAll(castDir, 0755)
		state.Checksum = hashStr
		state.Files[relFile] = hashStr
		if bytes, err := json.MarshalIndent(state, "", "  "); err == nil {
			os.WriteFile(stateFile, bytes, 0644)
		}
	}

	projectChecksumCache[p.Dir] = matches
	return matches
}
