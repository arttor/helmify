package imagePullSecrets

const helmExpression = "{{ .Values.imagePullSecrets | default list | toJson }}"

const ValuesHelp = `
## you can specify existing secrets to be useds as imagePullSecrets
# imagePullSecrets:
# - name: image-pull-secret
`

// ProcessSpecMap adds 'imagePullSecrets' to the podSpec in specMap, if it doesn't
// already has one defined.
func ProcessSpecMap(specMap map[string]interface{}) {

	if _, defined := specMap["imagePullSecrets"]; !defined {
		specMap["imagePullSecrets"] = helmExpression
	}

}
