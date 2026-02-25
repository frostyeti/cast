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

	castDir := filepath.Join(p.Dir, ".cast")
	stateFile := filepath.Join(castDir, "state.json")
	
	type State struct {
		Checksum string `json:"checksum"`
	}

	var state State
	matches := false

	if stateData, err := os.ReadFile(stateFile); err == nil {
		if json.Unmarshal(stateData, &state) == nil {
			if state.Checksum == hashStr {
				matches = true
			}
		}
	}

	if !matches {
		// Clear local cache for tasks if checksum doesn't match
		os.RemoveAll(filepath.Join(castDir, "tasks"))

		// Write new state
		os.MkdirAll(castDir, 0755)
		state.Checksum = hashStr
		if bytes, err := json.Marshal(state); err == nil {
			os.WriteFile(stateFile, bytes, 0644)
		}
	}

	projectChecksumCache[p.Dir] = matches
	return matches
}
