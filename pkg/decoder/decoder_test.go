package decoder

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

const (
	validObjects2 = `apiVersion: v1
kind: Service
metadata:
  name: my-operator-webhook-service
  namespace: my-operator-system
spec:
  ports:
  - port: 443
    targetPort: 9443
  selector:
    control-plane: controller-manager
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-system
`
	validObjects2withInvalid = `ajrcmq84xpru038um9q8
wqprux934ur8wcnqwp8urxqwrxuqweruncw
---
apiVersion: v1
kind: Service
metadata:
  name: my-operator-webhook-service
  namespace: my-operator-system
spec:
  ports:
  - port: 443
    targetPort: 9443
  selector:
    control-plane: controller-manager
---
---
---
8umx9284ru 82q983y49q
q 3408tuqw8e
q 49tuqw[fa iwfaowoewihfe4hf
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: my-operator-system
---apiVersion: v1
metadata:
  labels:
`
	validObjects0 = `---
---
---
`
)

func TestDecodeOk(t *testing.T) {
	reader := strings.NewReader(validObjects2)
	stop := make(chan struct{})
	objects := Decode(stop, reader)
	i := 0
	for _ = range objects {
		i++
	}
	assert.Equal(t, 2, i, "decoded two objects")
}

func TestDecodeEmptyObj(t *testing.T) {
	reader := strings.NewReader(validObjects0)
	stop := make(chan struct{})
	objects := Decode(stop, reader)
	i := 0
	for _ = range objects {
		i++
	}
	assert.Equal(t, 0, i, "decoded none objects")
}

func TestDecodeInvalidObj(t *testing.T) {
	reader := strings.NewReader(validObjects2withInvalid)
	stop := make(chan struct{})
	objects := Decode(stop, reader)
	i := 0
	for _ = range objects {
		i++
	}
	assert.Equal(t, 2, i, "decoded 2 valid objects")
}
