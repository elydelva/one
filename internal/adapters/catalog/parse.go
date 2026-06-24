package catalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"elydelva/one/internal/core"
	pkgcatalog "elydelva/one/pkg/catalog"
)

// parseServiceYAML decodes raw service.yaml bytes and validates the schema version.
// If def.ID is empty, fallbackID is used.
func parseServiceYAML(raw []byte, fallbackID string) (*pkgcatalog.ServiceDef, error) {
	var def pkgcatalog.ServiceDef
	if err := yaml.Unmarshal(raw, &def); err != nil {
		return nil, fmt.Errorf("parse service.yaml: %w", err)
	}
	if def.Version != 1 {
		return nil, fmt.Errorf("unsupported service.yaml version %d", def.Version)
	}
	if def.ID == "" {
		def.ID = fallbackID
	}
	return &def, nil
}

// parseActionYAML decodes a per-action YAML file. If def.ID is empty, fallbackID is used.
func parseActionYAML(raw []byte, fallbackID string) (*pkgcatalog.ActionDef, error) {
	var def pkgcatalog.ActionDef
	if err := yaml.Unmarshal(raw, &def); err != nil {
		return nil, err
	}
	if def.ID == "" {
		def.ID = fallbackID
	}
	return &def, nil
}

// buildService assembles a core.Service from a parsed ServiceDef plus
// optionally per-file action defs (which override inline actions on conflict).
func buildService(def *pkgcatalog.ServiceDef, fileActions []pkgcatalog.ActionDef) (*core.Service, error) {
	merged := map[string]pkgcatalog.ActionDef{}
	for id, a := range def.Actions {
		if a.ID == "" {
			a.ID = id
		}
		merged[id] = a
	}
	for _, a := range fileActions {
		merged[a.ID] = a
	}

	svc := &core.Service{
		ID:          core.ServiceID(def.ID),
		Name:        def.Name,
		Description: def.Description,
		BaseURL:     def.BaseURL,
		Providers:   parseProviders(def.Auth.Providers),
		Injection:   parseInjection(def.Auth.Injection),
		AuthConfigs: parseAuthConfigs(def.Auth.Config),
	}
	for _, action := range merged {
		ca, err := toCoreAction(svc.ID, action)
		if err != nil {
			return nil, fmt.Errorf("action %q: %w", action.ID, err)
		}
		svc.Actions = append(svc.Actions, ca)
	}
	sort.Slice(svc.Actions, func(i, j int) bool { return svc.Actions[i].ID < svc.Actions[j].ID })
	return svc, nil
}

// parseAuthConfigs maps the YAML auth.config block to per-provider core.AuthConfig.
func parseAuthConfigs(in map[string]pkgcatalog.ProviderConfigDef) map[core.ProviderKind]core.AuthConfig {
	if len(in) == 0 {
		return nil
	}
	out := map[core.ProviderKind]core.AuthConfig{}
	for k, v := range in {
		out[core.ProviderKind(k)] = core.AuthConfig{
			ClientID:       v.ClientID,
			AuthEndpoint:   v.AuthEndpoint,
			TokenEndpoint:  v.TokenEndpoint,
			DeviceEndpoint: v.DeviceEndpoint,
			Scopes:         append([]string(nil), v.Scopes...),
			RedirectPath:   v.RedirectPath,
			ValidateURL:    v.ValidateURL,
		}
	}
	return out
}

// parseGuideMarkdown decodes a guide markdown file with optional YAML frontmatter.
// Frontmatter format: leading `---\n<yaml>\n---\n<body>`. If absent, the whole
// file is treated as body and metadata is zero-valued.
func parseGuideMarkdown(raw []byte, id string, svc core.ServiceID) (*core.InstallGuide, error) {
	body := string(raw)
	meta := guideFrontmatter{}
	if strings.HasPrefix(body, "---\n") {
		rest := body[4:]
		end := strings.Index(rest, "\n---\n")
		if end == -1 {
			return nil, errors.New("unterminated frontmatter")
		}
		if err := yaml.Unmarshal([]byte(rest[:end]), &meta); err != nil {
			return nil, fmt.Errorf("parse frontmatter: %w", err)
		}
		body = rest[end+5:]
	}
	g := &core.InstallGuide{
		ID:      id,
		Service: svc,
		Title:   meta.Title,
		Content: body,
	}
	if meta.Verify != nil {
		g.Verify = &core.VerifyStep{
			Action: core.ActionID(meta.Verify.Action),
			Hint:   meta.Verify.Hint,
		}
	}
	return g, nil
}

// guideFrontmatter is the YAML schema for install-guide frontmatter.
// Other fields (auto_install, auto_detect_on_error) are surfaced separately
// in Slice D when the install use case is implemented.
type guideFrontmatter struct {
	Title             string      `yaml:"title"`
	AutoInstall       string      `yaml:"auto_install,omitempty"`
	AutoDetectOnError string      `yaml:"auto_detect_on_error,omitempty"`
	Verify            *verifyMeta `yaml:"verify,omitempty"`
}

type verifyMeta struct {
	Action string `yaml:"action"`
	Hint   string `yaml:"hint,omitempty"`
}
