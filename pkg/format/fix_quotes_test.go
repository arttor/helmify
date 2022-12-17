package format

import "testing"

func TestFixUnterminatedQuotes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "remove line break for unterminated quotes",
			in: `apiVersion: v1
kind: Secret
metadata:
  name: {{ include "app.fullname" . }}-my-secret-vars
  labels:
  {{- include "app.labels" . | nindent 4 }}
data:
  ELASTIC_FOOBAR_HUNTER123_MEOWTOWN_VERIFY: {{ required "mySecretVars.elasticFoobarHunter123MeowtownVerify
    is required" .Values.mySecretVars.elasticFoobarHunter123MeowtownVerify | b64enc
    | quote }}
  VAR1: {{ required "mySecretVars.var1 is required" .Values.mySecretVars.var1 | b64enc
    | quote }}
  VAR2: {{ required "mySecretVars.var2 is required" .Values.mySecretVars.var2 | b64enc
    | quote }}
stringData:
  str: {{ required "mySecretVars.str is required" .Values.mySecretVars.str | quote
    }}
type: opaque`,
			want: `apiVersion: v1
kind: Secret
metadata:
  name: {{ include "app.fullname" . }}-my-secret-vars
  labels:
  {{- include "app.labels" . | nindent 4 }}
data:
  ELASTIC_FOOBAR_HUNTER123_MEOWTOWN_VERIFY: {{ required "mySecretVars.elasticFoobarHunter123MeowtownVerify is required" .Values.mySecretVars.elasticFoobarHunter123MeowtownVerify | b64enc
    | quote }}
  VAR1: {{ required "mySecretVars.var1 is required" .Values.mySecretVars.var1 | b64enc
    | quote }}
  VAR2: {{ required "mySecretVars.var2 is required" .Values.mySecretVars.var2 | b64enc
    | quote }}
stringData:
  str: {{ required "mySecretVars.str is required" .Values.mySecretVars.str | quote
    }}
type: opaque`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FixUnterminatedQuotes(tt.in); got != tt.want {
				t.Errorf("FixUnterminatedQuotes() = %v, want %v", got, tt.want)
			}
		})
	}
}
