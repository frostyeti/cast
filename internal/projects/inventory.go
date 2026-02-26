package projects

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/eval"
	"github.com/frostyeti/cast/internal/paths"
	"github.com/frostyeti/cast/internal/types"
	"github.com/frostyeti/go/env"
	"go.yaml.in/yaml/v4"
)

func loadInventories(p *Project) error {
	if len(p.Schema.Inventories) == 0 {
		return nil
	}

	scope := p.Scope.ToMap()

	substitution := true
	if p.Schema.Config != nil && p.Schema.Config.Substitution != nil {
		substitution = *p.Schema.Config.Substitution
	}

	dataDir, err := paths.UserDataDir()
	if err != nil {
		return err
	}
	globalInvDir := filepath.Join(dataDir, "cast", "inventory")

	for _, invRef := range p.Schema.Inventories {
		var invPath string
		found := false

		// Try explicit path
		if filepath.IsAbs(invRef) || strings.HasPrefix(invRef, "./") || strings.HasPrefix(invRef, "../") {
			absPath := invRef
			if !filepath.IsAbs(invRef) {
				absPath = filepath.Join(p.Dir, invRef)
			}
			if _, err := os.Stat(absPath); err == nil {
				invPath = absPath
				found = true
			}
		}

		if !found {
			// Look in project CastDir (.cast/inventory)
			for _, ext := range []string{".yaml", ".yml"} {
				checkPath := filepath.Join(p.CastDir, "inventory", invRef+ext)
				if _, err := os.Stat(checkPath); err == nil {
					invPath = checkPath
					found = true
					break
				}

				// Try without adding extension if the user provided it
				if strings.HasSuffix(invRef, ".yaml") || strings.HasSuffix(invRef, ".yml") {
					checkPath = filepath.Join(p.CastDir, "inventory", invRef)
					if _, err := os.Stat(checkPath); err == nil {
						invPath = checkPath
						found = true
						break
					}
				}
			}
		}

		if !found {
			// Look in global directory
			for _, ext := range []string{".yaml", ".yml"} {
				checkPath := filepath.Join(globalInvDir, invRef+ext)
				if _, err := os.Stat(checkPath); err == nil {
					invPath = checkPath
					found = true
					break
				}

				if strings.HasSuffix(invRef, ".yaml") || strings.HasSuffix(invRef, ".yml") {
					checkPath = filepath.Join(globalInvDir, invRef)
					if _, err := os.Stat(checkPath); err == nil {
						invPath = checkPath
						found = true
						break
					}
				}
			}
		}

		if !found {
			return errors.Newf("inventory file not found: %s", invRef)
		}

		data, err := os.ReadFile(invPath)
		if err != nil {
			return errors.Newf("failed to read inventory %s: %v", invPath, err)
		}

		var inv types.Inventory
		err = yaml.Unmarshal(data, &inv)
		if err != nil {
			return errors.Newf("failed to parse inventory %s: %v", invPath, err)
		}

		if len(inv.Hosts) > 0 {
			defaultsMap := inv.Defaults

			// Iterate in HostOrder if available to preserve order, otherwise random
			var hostNames []string
			if len(inv.HostOrder) > 0 {
				hostNames = inv.HostOrder
			} else {
				for h := range inv.Hosts {
					hostNames = append(hostNames, h)
				}
			}

			for _, hostName := range hostNames {
				h := inv.Hosts[hostName]
				defaultsName := h.Defaults
				if defaultsName == "" {
					defaultsName = "default"
				}

				d, ok := defaultsMap[defaultsName]
				if ok {
					if (h.User == nil || *h.User == "") && d.User != nil {
						h.User = d.User
					}
					if (h.Port == nil || *h.Port == 0) && d.Port != nil {
						h.Port = d.Port
					}
					if (h.IdentityFile == nil || *h.IdentityFile == "") && d.IdentityFile != nil {
						h.IdentityFile = d.IdentityFile
					}
					if (h.Password == nil || *h.Password == "") && d.Password != nil {
						h.Password = d.Password
					}
					if h.OS == nil && d.OS != nil {
						h.OS = d.OS
					} else if h.OS != nil && d.OS != nil {
						if h.OS.Version == "" && d.OS.Version != "" {
							h.OS.Version = d.OS.Version
						}
						if h.OS.Arch == "" && d.OS.Arch != "" {
							h.OS.Arch = d.OS.Arch
						}
						if h.OS.Family == "" && d.OS.Family != "" {
							h.OS.Family = d.OS.Family
						}
						if h.OS.Variant == "" && d.OS.Variant != "" {
							h.OS.Variant = d.OS.Variant
						}
					}
					if h.Meta == nil && d.Meta != nil {
						h.Meta = d.Meta
					} else if h.Meta != nil && d.Meta != nil {
						for k, v := range d.Meta.ToMap() {
							if _, exists := h.Meta.ToMap()[k]; !exists {
								h.Meta.Set(k, v)
							}
						}
					}
					for _, g := range d.Tags {
						foundTag := false
						for _, t := range h.Tags {
							if t == g {
								foundTag = true
								break
							}
						}
						if !foundTag {
							h.Tags = append(h.Tags, g)
						}
					}
				} else {
					if defaultsName != "default" {
						return errors.Newf("host %s references undefined defaults %s in %s", h.Host, h.Defaults, invPath)
					}
				}

				if strings.ContainsRune(h.Host, '{') {
					v, err := eval.EvalAsString(h.Host, scope)
					if err != nil {
						return err
					}
					h.Host = v
				}

				if h.Password != nil && strings.ContainsRune(*h.Password, '{') {
					v, err := eval.EvalAsString(*h.Password, scope)
					if err != nil {
						return err
					}
					*h.Password = v
				}
				if h.Password != nil && strings.ContainsRune(*h.Password, '$') {
					v, err := env.Expand(*h.Password, env.WithGet(p.Env.Get), env.WithCommandSubstitution(substitution))
					if err != nil {
						return err
					}
					*h.Password = v
				}

				if h.IdentityFile != nil && strings.ContainsRune(*h.IdentityFile, '{') {
					v, err := eval.EvalAsString(*h.IdentityFile, scope)
					if err != nil {
						return err
					}
					*h.IdentityFile = v
				}
				if h.IdentityFile != nil && strings.ContainsRune(*h.IdentityFile, '$') {
					v, err := env.Expand(*h.IdentityFile, env.WithGet(p.Env.Get), env.WithCommandSubstitution(substitution))
					if err != nil {
						return err
					}
					*h.IdentityFile = v
				}

				for _, h2 := range p.Hosts {
					if h2.Host == h.Host {
						return errors.Newf("duplicate host entry for host %s in inventory %s", h.Host, invPath)
					}
				}

				port := uint(22)
				if h.Port != nil && *h.Port > 0 {
					port = *h.Port
				}

				user := ""
				if h.User != nil {
					user = *h.User
				}

				password := ""
				if h.Password != nil {
					password = *h.Password
				}

				identityFile := ""
				if h.IdentityFile != nil {
					identityFile = *h.IdentityFile
				}

				osInfo := types.OsInfo{}
				if h.OS != nil {
					osInfo = *h.OS
				}

				meta := types.NewMeta()
				if h.Meta != nil {
					meta = h.Meta
				}

				p.Hosts[hostName] = HostInfo{
					Host:         h.Host,
					Port:         port,
					User:         user,
					Password:     password,
					IdentityFile: identityFile,
					OS:           osInfo,
					Meta:         *meta,
					Tags:         h.Tags,
				}

				if _, ok := p.Hosts[h.Host]; !ok {
					p.Hosts[h.Host] = p.Hosts[hostName]
				}
			}
		}
	}
	return nil
}
