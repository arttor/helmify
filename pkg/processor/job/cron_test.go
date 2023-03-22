package job

import (
	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	strCron = `apiVersion: batch/v1
kind: CronJob
metadata:
  name: cron-job
spec:
  schedule: "* * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: hello
              image: busybox:1.28
              imagePullPolicy: IfNotPresent
              command:
                - /bin/sh
                - -c
                - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure`
)

func Test_Cron_Process(t *testing.T) {
	var testInstance cron

	t.Run("processed", func(t *testing.T) {
		obj := internal.GenerateObj(strCron)
		processed, _, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, true, processed)
	})
	t.Run("skipped", func(t *testing.T) {
		obj := internal.TestNs
		processed, _, err := testInstance.Process(&metadata.Service{}, obj)
		assert.NoError(t, err)
		assert.Equal(t, false, processed)
	})
}
