package decoder

import (
	"errors"
	"io"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	yamlDecoderBufferSize          = 100
	decoderResultChannelBufferSize = 1
)

// Decode - reads bytes stream of k8s objects in yaml format and decodes it to k8s unstructured objects.
// Non-blocking function. Sends results into buffered channel. Closes channel on io.EOF.
func Decode(stop <-chan struct{}, reader io.Reader) <-chan *unstructured.Unstructured {
	decoder := yamlutil.NewYAMLOrJSONDecoder(reader, yamlDecoderBufferSize)
	res := make(chan *unstructured.Unstructured, decoderResultChannelBufferSize)
	go func() {
		defer close(res)
		logrus.Debug("Start processing...")
		for {
			select {
			case <-stop:
				logrus.Debug("Exiting: received stop signal")
				return
			default:
			}
			var rawObj runtime.RawExtension
			err := decoder.Decode(&rawObj)
			if errors.Is(err, io.EOF) {
				logrus.Debug("EOF received. Finishing input objects decoding.")
				return
			}
			if err != nil {
				logrus.WithError(err).Error("unable to decode yaml from input")
				continue
			}
			obj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
			if err != nil {
				logrus.WithError(err).Error("unable to decode yaml")
				continue
			}
			unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
			if err != nil {
				logrus.WithError(err).Error("unable to map yaml to k8s unstructured")
				continue
			}
			object := &unstructured.Unstructured{Object: unstructuredMap}
			logrus.WithFields(logrus.Fields{
				"ApiVersion": object.GetAPIVersion(),
				"Kind":       object.GetKind(),
				"Name":       object.GetName(),
			}).Debug("decoded")
			res <- object
		}
	}()
	return res
}
