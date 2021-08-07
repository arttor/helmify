package decoder

import (
	"github.com/sirupsen/logrus"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	yamlDecoderBufferSize          = 100
	decoderResultChannelBufferSize = 1
)

func Decode(stop <-chan struct{}, reader io.Reader) <-chan *unstructured.Unstructured {
	decoder := yamlutil.NewYAMLOrJSONDecoder(reader, yamlDecoderBufferSize)
	res := make(chan *unstructured.Unstructured, decoderResultChannelBufferSize)
	go func(stop <-chan struct{}, reader io.Reader) {
		defer close(res)
		log := logrus.WithField("from", "decoder")
		for {
			select {
			case <-stop:
				return
			default:
			}
			var rawObj runtime.RawExtension
			err := decoder.Decode(&rawObj)
			if err == io.EOF {
				log.Debug("EOF received. Finishing input objects decoding.")
				return
			}
			if err != nil {
				log.WithError(err).Error("unable to decode yaml from input")
				return
			}
			obj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
			if err != nil {
				log.WithError(err).Error("unable to decode yaml")
				return
			}
			unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
			if err != nil {
				log.WithError(err).Error("unable to map yaml to k8s unstructured")
				return
			}
			object := &unstructured.Unstructured{Object: unstructuredMap}
			log.WithFields(logrus.Fields{
				"Kind": object.GetKind(),
				"Name": object.GetName(),
			}).Debug("decoded")
			res <- object
		}
	}(stop, reader)
	return res
}
