package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/arttor/helmify/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var actualExitCode int

func mockExit(code int) {
	actualExitCode = code
	panic("os.Exit called") // Panicking is necessary to stop execution.
}

func resetFlags(t *testing.T) {
	t.Helper()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestReadFlags_MutuallyExclusive(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine

	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})

	os.Args = []string{
		"helmify",
		"-crd-dir",
		"-optional-crds",
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	_, err := ReadFlags()
	require.Error(t, err)
	require.ErrorIs(t, err, errMutuallyExclusiveCRDs)
	require.Equal(t, errMutuallyExclusiveCRDs.Error(), err.Error())
}

func TestReadFlags_Version(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	oldOsExit := osExit
	stdout := os.Stdout

	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		osExit = oldOsExit
		os.Stdout = stdout
	})

	os.Args = []string{"helmify", "--version"}
	resetFlags(t)

	r, w, err := os.Pipe()
	require.NoError(t, err)

	osExit = mockExit
	os.Stdout = w

	var capturedOutput bytes.Buffer
	defer func() {
		require.NoError(t, w.Close())
		_, err = io.Copy(&capturedOutput, r)
		require.NoError(t, err)
		require.NoError(t, r.Close())
		require.NotNil(t, recover())

		expectedOutput := `Version:    development
Build Time: not set
Git Commit: not set
`
		assert.Equal(t, expectedOutput, capturedOutput.String())
		assert.Equal(t, 0, actualExitCode)
	}()
	_, err = ReadFlags()
	require.NoError(t, err)
}

func TestReadFlags_Help(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	oldOsExit := osExit
	stdout := os.Stdout

	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		osExit = oldOsExit
		os.Stdout = stdout
	})

	os.Args = []string{"helmify", "--help"}
	resetFlags(t)

	r, w, err := os.Pipe()
	require.NoError(t, err)

	osExit = mockExit
	os.Stdout = w

	var capturedOutput bytes.Buffer
	defer func() {
		require.NoError(t, w.Close())
		_, err = io.Copy(&capturedOutput, r)
		require.NoError(t, err)
		require.NoError(t, r.Close())
		require.NotNil(t, recover())

		var b strings.Builder
		b.WriteString(helpText)
		flag.CommandLine.SetOutput(&b)
		flag.PrintDefaults()

		assert.Equal(t, b.String(), capturedOutput.String())
		assert.Equal(t, 0, actualExitCode)
	}()
	_, err = ReadFlags()
	require.NoError(t, err)
}

func TestReadFlags_DefaultValuesMatchFlagDefaults(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine

	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})

	os.Args = []string{"helmify"}
	resetFlags(t)

	cfg, err := ReadFlags()
	require.NoError(t, err)

	stringTests := []struct {
		flagName string
		getValue func(cfg config.Config) string
	}{
		{
			flagName: "cert-manager-version",
			getValue: func(cfg config.Config) string { return cfg.CertManagerVersion },
		},
	}

	boolToStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}

	boolTests := []struct {
		flagName string
		getValue func(cfg config.Config) bool
	}{
		{"v", func(cfg config.Config) bool { return cfg.Verbose }},
		{"vv", func(cfg config.Config) bool { return cfg.VeryVerbose }},
		{"r", func(cfg config.Config) bool { return cfg.FilesRecursively }},

		{"crd-dir", func(cfg config.Config) bool { return cfg.Crd }},
		{"optional-crds", func(cfg config.Config) bool { return cfg.OptionalCRDs }},
		{"image-pull-secrets", func(cfg config.Config) bool { return cfg.ImagePullSecrets }},
		{"generate-defaults", func(cfg config.Config) bool { return cfg.GenerateDefaults }},
		{"cert-manager-as-subchart", func(cfg config.Config) bool { return cfg.CertManagerAsSubchart }},
		{"cert-manager-install-crd", func(cfg config.Config) bool { return cfg.CertManagerInstallCRD }},
		{"original-name", func(cfg config.Config) bool { return cfg.OriginalName }},
		{"preserve-ns", func(cfg config.Config) bool { return cfg.PreserveNs }},
		{"add-webhook-option", func(cfg config.Config) bool { return cfg.AddWebhookOption }},
	}

	for _, tt := range stringTests {
		t.Run("default_"+tt.flagName, func(t *testing.T) {
			f := flag.Lookup(tt.flagName)
			require.NotNil(t, f)
			assert.Equal(t, f.DefValue, tt.getValue(cfg))
		})
	}

	for _, tt := range boolTests {
		t.Run("default_"+tt.flagName, func(t *testing.T) {
			f := flag.Lookup(tt.flagName)
			require.NotNil(t, f)
			assert.Equal(t, f.DefValue, boolToStr(tt.getValue(cfg)))
		})
	}
}
