package translator

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Payload wraps an unstructured object with its source filename (if any).
type Payload struct {
	Object   *unstructured.Unstructured
	Filename string
}

// Translator defines the contract for turning a specific input source
// into a channel of k8s Unstructured objects that Helmify can process.
type Translator interface {
	Translate(ctx context.Context) (<-chan Payload, error)
}
