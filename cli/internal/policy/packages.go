package policy

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

//go:embed packages/*
var packagesFS embed.FS

// PackageManifest describes a fragment package.
type PackageManifest struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Inbound     []string `json:"inbound"`
	Outbound    []string `json:"outbound"`
}

// Package is a loaded fragment package ready for deployment.
type Package struct {
	Manifest  PackageManifest
	Fragments []Fragment
	fragMap   map[string]string
}

func (p *Package) AllFragmentIDs() []string {
	seen := make(map[string]bool)
	var ids []string
	for _, id := range p.Manifest.Inbound {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	for _, id := range p.Manifest.Outbound {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids
}

func (p *Package) InboundIDs() []string {
	return p.Manifest.Inbound
}

func (p *Package) OutboundIDs() []string {
	return p.Manifest.Outbound
}

func (p *Package) GetFragmentXML(id string) (string, bool) {
	xml, ok := p.fragMap[id]
	return xml, ok
}

func (p *Package) FragmentDescription(id string) string {
	if strings.TrimSpace(p.Manifest.Version) != "" {
		return fmt.Sprintf("HiddenLayer %s@%s fragment %s", p.Manifest.Name, p.Manifest.Version, id)
	}
	return fmt.Sprintf("HiddenLayer %s fragment %s", p.Manifest.Name, id)
}

func ListPackages() ([]string, error) {
	entries, err := fs.ReadDir(packagesFS, "packages")
	if err != nil {
		return nil, fmt.Errorf("reading packages directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		_, err := fs.Stat(packagesFS, path.Join("packages", e.Name(), "package.json"))
		if err == nil {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func LoadPackage(name string) (*Package, error) {
	manifest, err := LoadPackageManifest(name)
	if err != nil {
		return nil, err
	}

	pkg := &Package{
		Manifest: *manifest,
		fragMap:  make(map[string]string),
	}

	dir := path.Join("packages", name)
	entries, err := fs.ReadDir(packagesFS, dir)
	if err != nil {
		return nil, fmt.Errorf("reading package directory %q: %w", name, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		data, err := fs.ReadFile(packagesFS, path.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading fragment %q: %w", e.Name(), err)
		}
		id := strings.TrimSuffix(e.Name(), ".xml")
		xml := string(data)
		pkg.Fragments = append(pkg.Fragments, Fragment{ID: id, XML: xml})
		pkg.fragMap[id] = xml
	}

	return pkg, nil
}

func LoadPackageManifest(name string) (*PackageManifest, error) {
	data, err := fs.ReadFile(packagesFS, path.Join("packages", name, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("package %q not found: %w", name, err)
	}
	var m PackageManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest for %q: %w", name, err)
	}
	return &m, nil
}
