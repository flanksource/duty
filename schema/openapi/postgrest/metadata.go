package postgrest

import "github.com/getkin/kin-openapi/openapi3"

// Metadata is the Mission Control identity injected into every generated
// spec. The orchestrator calls ApplyMetadata after Convert so consumers see
// product-correct title/description/license/server URL instead of the
// PostgREST defaults ("standard public schema", localhost:<port>, …).
type Metadata struct {
	Title          string
	Description    string
	Version        string
	ContactName    string
	ContactURL     string
	ContactEmail   string
	LicenseName    string
	LicenseURL     string
	ServerURL      string
	ServerDesc     string
	DocsURL        string
	DocsDescription string
}

// DefaultMetadata returns the canonical Mission Control metadata. Callers
// can override individual fields before passing to ApplyMetadata.
func DefaultMetadata() Metadata {
	return Metadata{
		Title:           "Mission Control API",
		Description:     "Auto-generated PostgREST API for Flanksource Mission Control. The schema is regenerated from the live duty database via `task gen:openapi`; do not edit by hand.",
		Version:         "0.0.0",
		ContactName:     "Flanksource",
		ContactURL:      "https://flanksource.com",
		ContactEmail:    "hello@flanksource.com",
		LicenseName:     "Apache-2.0",
		LicenseURL:      "https://www.apache.org/licenses/LICENSE-2.0",
		ServerURL:       "https://{host}/db",
		ServerDesc:      "Mission Control PostgREST endpoint. Replace {host} with your installation's hostname.",
		DocsURL:         "https://docs.flanksource.com/mission-control/overview",
		DocsDescription: "Mission Control documentation",
	}
}

// ApplyMetadata replaces the spec's info, servers, and externalDocs blocks
// with the supplied Metadata. Empty fields are skipped so callers can
// partially override DefaultMetadata.
func ApplyMetadata(spec *openapi3.T, m Metadata) {
	if spec == nil {
		return
	}

	if spec.Info == nil {
		spec.Info = &openapi3.Info{}
	}
	if m.Title != "" {
		spec.Info.Title = m.Title
	}
	if m.Description != "" {
		spec.Info.Description = m.Description
	}
	if m.Version != "" {
		spec.Info.Version = m.Version
	}
	if m.ContactName != "" || m.ContactURL != "" || m.ContactEmail != "" {
		spec.Info.Contact = &openapi3.Contact{
			Name:  m.ContactName,
			URL:   m.ContactURL,
			Email: m.ContactEmail,
		}
	}
	if m.LicenseName != "" {
		spec.Info.License = &openapi3.License{
			Name: m.LicenseName,
			URL:  m.LicenseURL,
		}
	}

	if m.ServerURL != "" {
		spec.Servers = openapi3.Servers{{
			URL:         m.ServerURL,
			Description: m.ServerDesc,
		}}
	}

	if m.DocsURL != "" {
		spec.ExternalDocs = &openapi3.ExternalDocs{
			URL:         m.DocsURL,
			Description: m.DocsDescription,
		}
	}
}
