package server_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var TestCaCrt = `-----BEGIN CERTIFICATE-----
MIIBvzCCAWSgAwIBAgIRAKhVk049hVtC24ohZqzXSHAwCgYIKoZIzj0EAwIwTjEN
MAsGA1UEBhMEVGVzdDENMAsGA1UEBxMEVGVzdDENMAsGA1UEChMEVGVzdDENMAsG
A1UECxMEVGVzdDEQMA4GA1UEAxMHVGVzdCBDQTAeFw0yMDAyMDYxMTA1NTRaFw0z
MDAyMDMxMTA1NTRaME4xDTALBgNVBAYTBFRlc3QxDTALBgNVBAcTBFRlc3QxDTAL
BgNVBAoTBFRlc3QxDTALBgNVBAsTBFRlc3QxEDAOBgNVBAMTB1Rlc3QgQ0EwWTAT
BgcqhkjOPQIBBggqhkjOPQMBBwNCAAQ1JZwVjcOn0qxLr1rCQN5cYBdePoV+i2ie
ri+6dRt8JEqpR1+694+yWllCu+ldTlYVduU/pUOrUJ4oyYU3c6floyMwITAOBgNV
HQ8BAf8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAKBggqhkjOPQQDAgNJADBGAiEA
2xvxZ7EYxhUusLpZiKJmzKg2CZCAP4v8uzlI1JqePqACIQDJQlUwrVdARpC02v+J
3CcezG3lWHuMG1sTW4zekKuFiA==
-----END CERTIFICATE-----
`

var TestCrt = `-----BEGIN CERTIFICATE-----
MIIB2jCCAYGgAwIBAgIRAP5nV3phj3WbAHFiT/cY7vwwCgYIKoZIzj0EAwIwTjEN
MAsGA1UEBhMEVGVzdDENMAsGA1UEBxMEVGVzdDENMAsGA1UEChMEVGVzdDENMAsG
A1UECxMEVGVzdDEQMA4GA1UEAxMHVGVzdCBDQTAeFw0yMDAyMDYxMTA2MzZaFw0z
MDAyMDMxMTA2MzZaMC0xDTALBgNVBAYTBFRlc3QxDTALBgNVBAoTBFRlc3QxDTAL
BgNVBAMTBHRlc3QwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQn+5ei51r7pUNt
VKfn2rRsUsLROk0rDOQG9oEvzqjARiZwGEEumSkCdDV5MYpMYt0BmxX42dk8vXue
K3VxuI3ao2EwXzAjBgNVHREEHDAaggR0ZXN0ggxodHRwczovL3Rlc3SHBH8AAAEw
DAYDVR0TBAUwAwEBADALBgNVHQ8EBAMCA4gwHQYDVR0lBBYwFAYIKwYBBQUHAwIG
CCsGAQUFBwMBMAoGCCqGSM49BAMCA0cAMEQCIAOm/45P8C/njZZrs8iYEotOk3oQ
f7d8FwSKAagbNWomAiABQBEb9CvfG3so04yKmIMd/2XB5LXM2SQfBKdg/nMD8A==
-----END CERTIFICATE-----
`

var TestCrtKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIAqNQjvFqI95fIE/2UOMBM+mOJq0mCCkZTj/clWsa5VCoAoGCCqGSM49
AwEHoUQDQgAEJ/uXouda+6VDbVSn59q0bFLC0TpNKwzkBvaBL86owEYmcBhBLpkp
AnQ1eTGKTGLdAZsV+NnZPL17nit1cbiN2g==
-----END EC PRIVATE KEY-----
`

func TestValidateConfig(t *testing.T) {
	type args struct {
		configYaml       string
		caPoolIsOptional bool
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		want            server.Config
		wantCAPoolArray []string
	}{
		{
			name: "caPool as string",
			args: args{
				configYaml: `caPool: /tmp/test3570354545/ca3202122454
keyFile: /tmp/test3570354545/key3658110735
certFile: /tmp/test3570354545/crt4065348335
`,
			},
			want: server.Config{
				CAPool:   "/tmp/test3570354545/ca3202122454",
				KeyFile:  "/tmp/test3570354545/key3658110735",
				CertFile: "/tmp/test3570354545/crt4065348335",
			},
			wantCAPoolArray: []string{
				"/tmp/test3570354545/ca3202122454",
			},
		},
		{
			name: "caPool as []string",
			args: args{
				configYaml: `
caPool:
- /tmp/test3570354545/ca3202122454
keyFile: /tmp/test3570354545/key3658110735
certFile: /tmp/test3570354545/crt4065348335
`,
			},
			want: server.Config{
				CAPool: []interface{}{
					"/tmp/test3570354545/ca3202122454",
				},
				KeyFile:  "/tmp/test3570354545/key3658110735",
				CertFile: "/tmp/test3570354545/crt4065348335",
			},
			wantCAPoolArray: []string{
				"/tmp/test3570354545/ca3202122454",
			},
		},
		{
			name: "caPool is not set",
			args: args{
				configYaml: `
keyFile: /tmp/test3570354545/key3658110735
certFile: /tmp/test3570354545/crt4065348335
`,
				caPoolIsOptional: true,
			},
			want: server.Config{
				KeyFile:          "/tmp/test3570354545/key3658110735",
				CertFile:         "/tmp/test3570354545/crt4065348335",
				CAPoolIsOptional: true,
			},
		},
		{
			name: "caPool is not set - error",
			args: args{
				configYaml: `
keyFile: /tmp/test3570354545/key3658110735
certFile: /tmp/test3570354545/crt4065348335
`,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testCfg server.Config
			err := yaml.Unmarshal([]byte(tt.args.configYaml), &testCfg)
			require.NoError(t, err)
			testCfg.CAPoolIsOptional = tt.args.caPoolIsOptional
			err = testCfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			err = tt.want.Validate()
			require.NoError(t, err)
			require.Equal(t, tt.want, testCfg)
			caPoolArray, err := testCfg.CAPoolArray()
			require.NoError(t, err)
			require.Equal(t, tt.wantCAPoolArray, caPoolArray)
		})
	}
}

func TestNew(t *testing.T) {
	// tmp dir
	tmpDir, err := os.MkdirTemp("/tmp", "test")
	require.NoError(t, err)
	defer func() {
		_ = deleteTmpDir(tmpDir)
	}()
	// ca
	caFile, err := os.CreateTemp(tmpDir, "ca")
	require.NoError(t, err)
	err = caFile.Close()
	require.NoError(t, err)

	crtFile, err := os.CreateTemp(tmpDir, "crt")
	require.NoError(t, err)
	err = crtFile.Close()
	require.NoError(t, err)

	keyFile, err := os.CreateTemp(tmpDir, "key")
	require.NoError(t, err)
	err = keyFile.Close()
	require.NoError(t, err)

	config := createTmpCertFiles(t, caFile.Name(), crtFile.Name(), keyFile.Name())
	err = config.Validate()
	require.NoError(t, err)

	configYaml, err := yaml.Marshal(config)
	require.NoError(t, err)

	fmt.Printf("%s\n", configYaml)

	var testCfg server.Config
	err = yaml.Unmarshal(configYaml, &testCfg)
	require.NoError(t, err)

	err = testCfg.Validate()
	require.NoError(t, err)

	logger := log.NewLogger(log.MakeDefaultConfig())
	// cert manager
	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	mng, err := server.New(config, fileWatcher, logger)
	require.NoError(t, err)

	tlsConfig := mng.GetTLSConfig()
	require.NotNil(t, tlsConfig.GetCertificate)
	firstCrt, err := tlsConfig.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, firstCrt)

	// delete crt/key files
	deleteTmpCertFiles(t, config)
	// create new crt/key files
	createTmpCertFiles(t, caFile.Name(), crtFile.Name(), keyFile.Name())
	tlsConfig = mng.GetTLSConfig()

	require.NotNil(t, tlsConfig.GetCertificate)
	secondCrt, err := tlsConfig.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, secondCrt)

	require.Equal(t, firstCrt.Certificate, secondCrt.Certificate)
}

func createTmpCertFiles(t *testing.T, caFile, crtFile, keyFile string) server.Config {
	// ca
	err := os.WriteFile(caFile, []byte(TestCaCrt), os.FileMode(os.O_RDWR))
	require.NoError(t, err)

	// crt
	err = os.WriteFile(crtFile, []byte(TestCrt), os.FileMode(os.O_RDWR))
	require.NoError(t, err)

	// key
	err = os.WriteFile(keyFile, []byte(TestCrtKey), os.FileMode(os.O_RDWR))
	require.NoError(t, err)

	cfg := server.Config{
		CAPool:   caFile,
		KeyFile:  keyFile,
		CertFile: crtFile,
	}
	return cfg
}

func deleteTmpCertFiles(t *testing.T, cfg server.Config) {
	caPoolArray, error := cfg.CAPoolArray()
	require.NoError(t, error)
	for _, ca := range caPoolArray {
		err := os.Remove(ca)
		require.NoError(t, err)
	}
	err := os.Remove(cfg.CertFile)
	require.NoError(t, err)
	err = os.Remove(cfg.KeyFile)
	require.NoError(t, err)
}

func deleteTmpDir(tmpDir string) error {
	return os.RemoveAll(tmpDir)
}
