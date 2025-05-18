package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateData contains the individual variables required by package.json.tmpl.
// We purposefully expand the fields instead of using slices/maps because the
// template explicitly references them one by one.
type TemplateData struct {
	Version     string
	Description string

	DarwinArm64Name     string
	DarwinArm64URL      string
	DarwinArm64Checksum string

	DarwinX64Name     string
	DarwinX64URL      string
	DarwinX64Checksum string

	LinuxArm64Name     string
	LinuxArm64URL      string
	LinuxArm64Checksum string

	LinuxX64Name     string
	LinuxX64URL      string
	LinuxX64Checksum string

	Win32Arm64Name     string
	Win32Arm64URL      string
	Win32Arm64Checksum string

	Win32X64Name     string
	Win32X64URL      string
	Win32X64Checksum string
}

// Metadata mirrors the subset of fields we care about from metadata.json.
type Metadata struct {
	ProjectName string `json:"project_name"`
	Version     string `json:"version"`
}

// BrewConfig describes the data nested inside the Brew Tap artifact.
type BrewConfig struct {
	Description string `json:"description"`
	URLTemplate string `json:"url_template"`
}

type brewTapExtra struct {
	BrewConfig BrewConfig `json:"BrewConfig"`
}

// Artifact represents a single element of artifacts.json.
type Artifact struct {
	Name   string          `json:"name"`
	Path   string          `json:"path"`
	GOOS   string          `json:"goos"`
	GOARCH string          `json:"goarch"`
	Type   string          `json:"type"`
	Extra  json.RawMessage `json:"extra"`
}

type archiveExtra struct {
	Checksum string `json:"Checksum"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/npmpublish/npmpublish.go <tmpl_dir> <dist_dir>")
		os.Exit(1)
	}

	tmplDir := os.Args[1]
	distDir := os.Args[2]

	metadata, err := loadMetadata(filepath.Join(distDir, "metadata.json"))
	must(err)

	artifacts, brewCfg, err := loadArtifacts(filepath.Join(distDir, "artifacts.json"))
	must(err)

	data, err := buildTemplateData(metadata, brewCfg, artifacts)
	must(err)

	// Prepare .npmdist directory.
	outDir := ".npmdist"
	_ = os.RemoveAll(outDir) // ignore error if it doesn't exist
	must(copyDir(tmplDir, outDir))

	tmplPath := filepath.Join(outDir, "package.json.tmpl")
	outPackageJSON := filepath.Join(outDir, "package.json")

	must(renderTemplate(tmplPath, outPackageJSON, data))
	must(os.Remove(tmplPath))

	// Copy README.md to .npmdist directory
	readmeDestPath := filepath.Join(outDir, "README.md")
	must(copyFile("README.md", readmeDestPath, 0644))

	fmt.Println("npm publish ready at .npmdist")
}

func loadMetadata(path string) (*Metadata, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Metadata
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func loadArtifacts(path string) ([]Artifact, BrewConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, BrewConfig{}, err
	}
	var artifacts []Artifact
	if err := json.Unmarshal(b, &artifacts); err != nil {
		return nil, BrewConfig{}, err
	}

	var brewCfg BrewConfig
	found := false

	for _, a := range artifacts {
		if a.Type == "Brew Tap" {
			var ext brewTapExtra
			if err := json.Unmarshal(a.Extra, &ext); err != nil {
				return nil, BrewConfig{}, err
			}
			brewCfg = ext.BrewConfig
			found = true
			break
		}
	}
	if !found {
		return nil, BrewConfig{}, errors.New("Brew Tap artifact not found in artifacts.json")
	}

	return artifacts, brewCfg, nil
}

func buildTemplateData(meta *Metadata, brewCfg BrewConfig, artifacts []Artifact) (*TemplateData, error) {
	data := &TemplateData{
		Version:     meta.Version,
		Description: brewCfg.Description,
	}

	urlTmpl, err := template.New("url").Parse(brewCfg.URLTemplate)
	if err != nil {
		return nil, err
	}

	fill := func(goos, goarch string) (string, error) {
		var buf bytes.Buffer
		cfg := map[string]string{
			"ProjectName": meta.ProjectName,
			"Version":     meta.Version,
			"Os":          goos,
			"Arch":        goarch,
		}
		if err := urlTmpl.Execute(&buf, cfg); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	for _, a := range artifacts {
		if a.Type != "Archive" {
			continue
		}
		var ext archiveExtra
		if err := json.Unmarshal(a.Extra, &ext); err != nil {
			return nil, err
		}
		digest := strings.TrimPrefix(ext.Checksum, "sha256:")
		url, err := fill(a.GOOS, a.GOARCH)
		if err != nil {
			return nil, err
		}

		switch a.GOOS {
		case "darwin":
			if a.GOARCH == "arm64" {
				data.DarwinArm64Name = a.Name
				data.DarwinArm64Checksum = digest
				data.DarwinArm64URL = url
			} else if a.GOARCH == "amd64" {
				data.DarwinX64Name = a.Name
				data.DarwinX64Checksum = digest
				data.DarwinX64URL = url
			}
		case "linux":
			if a.GOARCH == "arm64" {
				data.LinuxArm64Name = a.Name
				data.LinuxArm64Checksum = digest
				data.LinuxArm64URL = url
			} else if a.GOARCH == "amd64" {
				data.LinuxX64Name = a.Name
				data.LinuxX64Checksum = digest
				data.LinuxX64URL = url
			}
		case "windows":
			if a.GOARCH == "arm64" {
				data.Win32Arm64Name = a.Name
				data.Win32Arm64Checksum = digest
				data.Win32Arm64URL = url
			} else if a.GOARCH == "amd64" {
				data.Win32X64Name = a.Name
				data.Win32X64Checksum = digest
				data.Win32X64URL = url
			}
		}
	}

	return data, nil
}

func renderTemplate(tmplPath, outPath string, data *TemplateData) error {
	t, err := template.ParseFiles(tmplPath)
	if err != nil {
		return err
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	if err := t.Execute(outFile, data); err != nil {
		return err
	}
	return nil
}

// copyDir recursively copies a directory tree, attempting to preserve permissions.
func copyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		// Copy file
		if err := copyFile(path, target, info.Mode()); err != nil {
			return err
		}
		return nil
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return os.Chmod(dst, mode)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
