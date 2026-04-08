package k8smanifest

import (
	"context"
	"io"

	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/decoder"
	"github.com/arttor/helmify/pkg/file"
	"github.com/arttor/helmify/pkg/translator"
)

// Translator implements translator.Translator for raw Kubernetes YAML/JSON manifests.
type Translator struct {
	config config.Config
	stdin  io.Reader
}

// New creates a new k8smanifest Translator.
func New(conf config.Config, stdin io.Reader) translator.Translator {
	return &Translator{
		config: conf,
		stdin:  stdin,
	}
}

// Translate reads files or stdin as configured and yields Unstructured objects.
func (t *Translator) Translate(ctx context.Context) (<-chan translator.Payload, error) {
	out := make(chan translator.Payload)

	go func() {
		defer close(out)

		if len(t.config.Files) != 0 {
			file.Walk(t.config.Files, t.config.FilesRecursively, func(filename string, fileReader io.Reader) {
				objects := decoder.Decode(ctx.Done(), fileReader)
				for obj := range objects {
					select {
					case <-ctx.Done():
						return
					case out <- translator.Payload{Object: obj, Filename: filename}:
					}
				}
			})
		} else {
			objects := decoder.Decode(ctx.Done(), t.stdin)
			for obj := range objects {
				select {
				case <-ctx.Done():
					return
				case out <- translator.Payload{Object: obj, Filename: ""}:
				}
			}
		}
	}()

	return out, nil
}
