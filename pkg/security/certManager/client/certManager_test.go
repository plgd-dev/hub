package client_test

import (
	"os"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	testX509 "github.com/plgd-dev/hub/v2/test/security/x509"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"gopkg.in/yaml.v3"
)

func TestValidateConfig(t *testing.T) {
	type args struct {
		configYaml string
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		want            client.Config
		wantCAPoolArray []urischeme.URIScheme
	}{
		{
			name: "caPool as string",
			args: args{
				configYaml: `caPool: /tmp/test3570354545/ca3202122454
keyFile: /tmp/test3570354545/key3658110735
certFile: /tmp/test3570354545/crt4065348335
`,
			},
			want: client.Config{
				CAPool:   "/tmp/test3570354545/ca3202122454",
				KeyFile:  "/tmp/test3570354545/key3658110735",
				CertFile: "/tmp/test3570354545/crt4065348335",
			},
			wantCAPoolArray: []urischeme.URIScheme{
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
			want: client.Config{
				CAPool:   []interface{}{"/tmp/test3570354545/ca3202122454"},
				KeyFile:  "/tmp/test3570354545/key3658110735",
				CertFile: "/tmp/test3570354545/crt4065348335",
			},
			wantCAPoolArray: []urischeme.URIScheme{
				"/tmp/test3570354545/ca3202122454",
			},
		},
		{
			name: "caPool is not set",
			args: args{
				configYaml: `
keyFile: /tmp/test3570354545/key3658110735
certFile: /tmp/test3570354545/crt4065348335
useSystemCAPool: true
`,
			},
			want: client.Config{
				KeyFile:         "/tmp/test3570354545/key3658110735",
				CertFile:        "/tmp/test3570354545/crt4065348335",
				UseSystemCAPool: true,
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
			var config client.Config
			err := yaml.Unmarshal([]byte(tt.args.configYaml), &config)
			require.NoError(t, err)
			err = config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			err = tt.want.Validate()
			require.NoError(t, err)
			require.Equal(t, tt.want, config)
			caPoolArray, err := config.CAPoolArray()
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

	logger := log.NewLogger(log.MakeDefaultConfig())
	// cert manager
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()
	mng, err := client.New(config, fileWatcher, logger, noop.NewTracerProvider())
	require.NoError(t, err)

	tlsConfig := mng.GetTLSConfig()
	require.NotNil(t, tlsConfig.GetClientCertificate)
	firstCrt, err := tlsConfig.GetClientCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, firstCrt)

	// delete crt/key files
	deleteTmpCertFiles(t, config)
	// create new crt/key files
	createTmpCertFiles(t, caFile.Name(), crtFile.Name(), keyFile.Name())
	// to be sure that file watcher will detect changes
	time.Sleep(time.Second * 4)
	tlsConfig = mng.GetTLSConfig()

	require.NotNil(t, tlsConfig.GetClientCertificate)
	secondCrt, err := tlsConfig.GetClientCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, secondCrt)

	require.NotEqual(t, firstCrt.Certificate, secondCrt.Certificate)
}

func createTmpCertFiles(t *testing.T, caFile, crtFile, keyFile string) client.Config {
	caPEM, caPrivKey := testX509.CreateCACertificate(t)
	inCAPEM, inCAPrivKey := testX509.CreateIntermediateCACertificate(t, caPEM, caPrivKey)

	// ca
	err := os.WriteFile(caFile, caPEM, 0o600)
	require.NoError(t, err)

	// crt
	err = os.WriteFile(crtFile, inCAPEM, 0o600)
	require.NoError(t, err)

	// key
	err = os.WriteFile(keyFile, testX509.PrivateKeyToPem(t, inCAPrivKey), 0o600)
	require.NoError(t, err)

	cfg := client.Config{
		CAPool:   caFile,
		KeyFile:  urischeme.URIScheme(keyFile),
		CertFile: urischeme.URIScheme(crtFile),
	}
	return cfg
}

func deleteTmpCertFiles(t *testing.T, cfg client.Config) {
	caPoolArray, err := cfg.CAPoolArray()
	require.NoError(t, err)
	for _, ca := range caPoolArray {
		err = os.Remove(ca.FilePath())
		require.NoError(t, err)
	}
	err = os.Remove(cfg.CertFile.FilePath())
	require.NoError(t, err)
	err = os.Remove(cfg.KeyFile.FilePath())
	require.NoError(t, err)
}

func deleteTmpDir(tmpDir string) error {
	return os.RemoveAll(tmpDir)
}
