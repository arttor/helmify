package helm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"path/filepath"
	"strings"

	"github.com/arttor/helmify/pkg/cluster"
	"github.com/arttor/helmify/pkg/helmify"
)

// MemoryOutput captures the generated Helm chart in memory.
// It implements the helmify.Output interface.
type MemoryOutput struct {
	Files map[string][]byte
}

// NewMemoryOutput creates a new MemoryOutput.
func NewMemoryOutput() *MemoryOutput {
	return &MemoryOutput{
		Files: make(map[string][]byte),
	}
}

func (m *MemoryOutput) Create(chartDir, chartName string, crd bool, certManagerAsSubchart bool, certManagerVersion string, certManagerInstallCRD bool, templates []helmify.Template, filenames []string) error {
	m.Files["Chart.yaml"] = chartYAML(chartName, certManagerAsSubchart, certManagerVersion)
	m.Files[".helmignore"] = []byte(helmIgnore)
	m.Files[filepath.Join("templates", "_helpers.tpl")] = helpersYAML(chartName)

	// Group templates into files
	files := map[string][]helmify.Template{}
	values := helmify.Values{}
	values[cluster.DomainKey] = cluster.DefaultDomain

	for i, template := range templates {
		file := files[filenames[i]]
		file = append(file, template)
		files[filenames[i]] = file
		if err := values.Merge(template.Values()); err != nil {
			return err
		}
	}

	// Write templates to memory
	for filename, tpls := range files {
		var subdir string
		if strings.Contains(filename, "crd") && crd {
			subdir = "crds"
		} else {
			subdir = "templates"
		}
		
		var buf bytes.Buffer
		for i, t := range tpls {
			if err := t.Write(&buf); err != nil {
				return err
			}
			if i != len(tpls)-1 {
				buf.Write([]byte("\n---\n"))
			}
		}
		if len(tpls) != 0 {
			buf.Write([]byte("\n"))
		}
		m.Files[filepath.Join(subdir, filename)] = buf.Bytes()
	}

	// Write values.yaml to memory
	if certManagerAsSubchart {
		_, _ = values.Add(certManagerInstallCRD, "certmanager", "installCRDs")
		_, _ = values.Add(true, "certmanager", "enabled")
	}
	res, err := marshalOrdered(values)
	if err != nil {
		return err
	}
	m.Files["values.yaml"] = res

	return nil
}

// ToTarGz bundles the captured files into a tar.gz stream.
func (m *MemoryOutput) ToTarGz(chartName string, w io.Writer) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for name, content := range m.Files {
		// All files should be nested inside a directory with the chart name
		path := filepath.Join(chartName, name)
		header := &tar.Header{
			Name: path,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tw.Write(content); err != nil {
			return err
		}
	}
	return nil
}
