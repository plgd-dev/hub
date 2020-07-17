package refImpl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/service"
	storeMongodb "github.com/go-ocf/cloud/cloud2cloud-gateway/store/mongodb"
	"github.com/go-ocf/kit/log"
	kitNetHttp "github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/kit/security/certManager"
)

type Config struct {
	Log          log.Config `envconfig:"LOG"`
	Service      service.Config
	StoreMongoDB storeMongodb.Config
	Dial         certManager.Config `envconfig:"DIAL"`
	Listen       certManager.Config `envconfig:"LISTEN"`
	JwksURL      string             `envconfig:"JWKS_URL"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

type RefImpl struct {
	server            *service.Server
	dialCertManager   certManager.CertManager
	listenCertManager certManager.CertManager
}

func Init(config Config) (*RefImpl, error) {
	log.Setup(config.Log)

	dialCertManager, err := certManager.NewCertManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create dial cert manager %w", err)
	}
	dialTLSConfig := dialCertManager.GetClientTLSConfig()

	substore, err := storeMongodb.NewStore(context.Background(), config.StoreMongoDB, storeMongodb.WithTLS(dialTLSConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create mongodb substore %w", err)
	}

	listenCertManager, err := certManager.NewCertManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create listen cert manager %w", err)
	}

	log.Info(config.String())

	auth := kitNetHttp.NewInterceptor(config.JwksURL, dialCertManager.GetClientTLSConfig(), authRules)

	return &RefImpl{
		server:            service.New(config.Service, dialCertManager, listenCertManager, auth, substore),
		dialCertManager:   dialCertManager,
		listenCertManager: listenCertManager,
	}, nil
}

func (r *RefImpl) Serve() error {
	return r.server.Serve()
}

func (r *RefImpl) Close() {
	r.server.Shutdown()
	r.dialCertManager.Close()
	r.listenCertManager.Close()
}

// https://openconnectivity.org/draftspecs/Gaborone/OCF_Cloud_API_for_Cloud_Services.pdf
var authRules = map[string][]kitNetHttp.AuthArgs{
	http.MethodGet: {
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]*\?` + service.ContentQuery + `=` + service.ContentQueryBaseValue + `$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]*\?` + service.ContentQuery + `=` + service.ContentQueryAllValue + `$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`r:resources:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*\?` + service.ContentQuery + `=` + service.ContentQueryBaseValue + `$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*\?` + service.ContentQuery + `=` + service.ContentQueryAllValue + `$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`r:resources:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+.*[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:resources:.*`),
			},
		},
	},
	http.MethodPost: {
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+subscriptions[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+subscriptions[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+.*[\/]+subscriptions[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:resources:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+.*[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`w:resources:.*`),
			},
		},
	},
	http.MethodDelete: {
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+subscriptions[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+subscriptions[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+.*[\/]+subscriptions[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:resources:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
	},
}
