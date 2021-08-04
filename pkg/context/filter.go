package context

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

func (c *Context) filter(obj *unstructured.Unstructured) bool {
	if c.config.ProcessOnly == nil && len(c.config.ProcessOnly) == 0 {
		return true
	}
	//TODO: implement filter
	return true
}
