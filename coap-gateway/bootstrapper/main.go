package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/google/uuid"
	cmV1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmMetaV1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certManager "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	cmInformers "github.com/jetstack/cert-manager/pkg/client/informers/externalversions"
	cfgLoader "github.com/plgd-dev/cloud/pkg/config"
	"github.com/plgd-dev/kit/security/generateCertificate"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Configuration struct {
	CloudID           string                            `yaml:"cloudID"`
	SecretName        string                            `yaml:"secretName"`
	IssuerName        string                            `yaml:"issuerName"`
	CertFileName      string                            `yaml:"certFileName"`
	KeyFileName       string                            `yaml:"keyFileName"`
	CAFileName        string                            `yaml:"caFileName"  `
	IncludeCA         bool                              `yaml:"includeCA"`
	CertConfiguration generateCertificate.Configuration `yaml:"certConfiguration"`
}

type Bootstrapper struct {
	configuration     *Configuration
	namespace         string
	kubernetesClient  *kubernetes.Clientset
	certManagerClient *certManager.Clientset
}

func main() {
	var bs Bootstrapper
	bs.loadConfiguration()
	bs.initializeClients()
	csr, pkeyBlock := bs.generateCSR()

	certRequest := cmV1.CertificateRequest{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      bs.configuration.SecretName,
			Namespace: bs.namespace,
		},
		Spec: cmV1.CertificateRequestSpec{
			Request: csr,
			IsCA:    false,
			IssuerRef: cmMetaV1.ObjectReference{
				Name: bs.configuration.IssuerName,
			},
		},
	}

	bs.cleanupCertificateRequest()
	factory := cmInformers.NewSharedInformerFactory(bs.certManagerClient, 0)
	informer := factory.Certmanager().V1().CertificateRequests().Informer()
	informerStopCh := make(chan struct{})

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			certRequest := obj.(*cmV1.CertificateRequest)
			if bs.isCertificateReady(certRequest) {
				bs.createSecret(pkeyBlock, certRequest.Status.Certificate, certRequest.Status.CA)
				close(informerStopCh)
			}
		},
		UpdateFunc: func(obj interface{}, newObj interface{}) {
			certRequest := newObj.(*cmV1.CertificateRequest)
			if bs.isCertificateReady(certRequest) {
				bs.createSecret(pkeyBlock, certRequest.Status.Certificate, certRequest.Status.CA)
				bs.cleanupCertificateRequest()
				close(informerStopCh)
			}
		},
	})
	go informer.Run(informerStopCh)

	_, err := bs.certManagerClient.CertmanagerV1().CertificateRequests(bs.namespace).Create(context.TODO(), &certRequest, metaV1.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}

	<-informerStopCh
}

func (bs *Bootstrapper) loadConfiguration() {
	bs.configuration = &Configuration{}
	err := cfgLoader.LoadAndValidateConfig(bs.configuration)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	namespace, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatalf("cannot get current namespace: %v", err)
	}
	bs.namespace = strings.TrimSuffix(string(namespace), "\n")
}

func (c *Configuration) Validate() error {
	if c.CloudID == "" {
		return fmt.Errorf("cloudID is required")
	}
	if _, err := uuid.Parse(c.CloudID); err != nil {
		return fmt.Errorf("cloudID('%v')", c.CloudID)
	}
	if c.SecretName == "" {
		return fmt.Errorf("secretName('%v')", c.SecretName)
	}
	return nil
}

func (bs *Bootstrapper) initializeClients() {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	kubernetesClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	bs.kubernetesClient = kubernetesClient

	certManagerClient, err := certManager.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	bs.certManagerClient = certManagerClient
}

func (bs *Bootstrapper) generateCSR() (pkeyPEM []byte, csr []byte) {
	pkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	pkeyBytes, err := x509.MarshalECPrivateKey(pkey)
	if err != nil {
		log.Fatal(err)
	}

	csrBytes, err := generateCertificate.GenerateIdentityCSR(bs.configuration.CertConfiguration, bs.configuration.CloudID, pkey)
	if err != nil {
		log.Fatal(err)
	}

	pkeyBlock := &pem.Block{Type: "EC PRIVATE KEY", Bytes: pkeyBytes}
	return csrBytes, pem.EncodeToMemory(pkeyBlock)
}

func (bs *Bootstrapper) cleanupCertificateRequest() {
	err := bs.certManagerClient.CertmanagerV1().CertificateRequests(bs.namespace).Delete(context.TODO(), bs.configuration.SecretName, metaV1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Fatalf("unable to delete certificate request: %v", err)
	}
}

func (bs *Bootstrapper) isCertificateReady(certRequest *cmV1.CertificateRequest) bool {
	if certRequest.Namespace == bs.namespace && certRequest.Name == bs.configuration.SecretName {
		latestStatusIndex := len(certRequest.Status.Conditions) - 1
		if latestStatusIndex >= 0 {
			latestStatus := certRequest.Status.Conditions[latestStatusIndex]
			log.Printf("status of certificate request %v updated: %v", bs.configuration.SecretName, latestStatus)
			return latestStatus.Type == cmV1.CertificateRequestConditionReady && latestStatus.Status == cmMetaV1.ConditionTrue
		}
	}
	return false
}

func (bs *Bootstrapper) createSecret(pkeyBlock, cert, ca []byte) {
	block, _ := pem.Decode(cert)
	if block == nil {
		log.Fatalf("failed to decode pem block: %v", string(cert))
	}
	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatalf("failed to parse certificate: %v", err)
	}
	log.Printf("certificate valid not after %v; not before %v", parsedCert.NotAfter, parsedCert.NotBefore)

	secretSpec := coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name: bs.configuration.SecretName,
		},
		Data: map[string][]byte{
			"tls.crt": cert,
			"tls.key": pkeyBlock,
			"ca.crt":  ca,
		},
	}

	_, err = bs.kubernetesClient.CoreV1().Secrets(bs.namespace).Get(context.TODO(), bs.configuration.SecretName, metaV1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = bs.kubernetesClient.CoreV1().Secrets(bs.namespace).Create(context.TODO(), &secretSpec, metaV1.CreateOptions{})
		if err != nil {
			log.Fatalf("unable to create secret: %v", err)
		}
		log.Printf("secret %v created", bs.configuration.SecretName)
	} else if err == nil {
		_, err = bs.kubernetesClient.CoreV1().Secrets(bs.namespace).Update(context.TODO(), &secretSpec, metaV1.UpdateOptions{})
		if err != nil {
			log.Fatalf("unable to update secret: %v", err)
		}
		log.Printf("secret %v updated", bs.configuration.SecretName)
	} else {
		log.Fatalf("unable to verify if secret %v exists: %v", bs.configuration.SecretName, err)
	}
}
