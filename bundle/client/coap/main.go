package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/plgd-dev/device/v2/schema/account"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/schema/session"
	"github.com/plgd-dev/go-coap/v3/message"
	codes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/options"
	coap "github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/go-coap/v3/tcp/client"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/plgd-dev/kit/v2/net"
)

type authReq struct {
	Accesstoken  string `json:"accesstoken"`
	DeviceID     string `json:"di"`
	AuthProvider string `json:"authprovider"`
}

type authResp struct {
	Accesstoken string `json:"accesstoken"`
	UID         string `json:"uid"`
	DeviceID    string `json:"di"`
	Login       bool   `json:"login"`
}

func signUp(co *client.Conn, authreq authReq) authResp {
	bw := bytes.NewBuffer(make([]byte, 0, 1024))
	err := cbor.WriteTo(bw, authreq)
	if err != nil {
		log.Fatalf("cannt encode signup req: %v", err)
	}

	resp, err := co.Post(context.Background(), account.ResourceURI, message.AppCBOR, bytes.NewReader(bw.Bytes()))
	if err != nil {
		log.Fatalf("error sending request to signup: %v", err)
	}
	if resp.Code() != codes.Changed {
		log.Fatalf("error get coap code for signup: %v", resp.Code())
	}

	var authresp authResp
	err = cbor.ReadFrom(resp.Body(), &authresp)
	if err != nil {
		log.Fatalf("cannot decode authresp: %v", err)
	}

	return authresp
}

func signUpWithAuthCode(co *client.Conn, authCode, deviceID string) (accessToken, uid string) {
	authreq := authReq{
		Accesstoken:  authCode,
		DeviceID:     deviceID,
		AuthProvider: "plgd",
	}
	authresp := signUp(co, authreq)
	authresp.DeviceID = deviceID
	authresp.Login = true
	return authresp.Accesstoken, authresp.UID
}

func signIn(co *client.Conn, authresp authResp) {
	log.Printf("authresp: \n%v\n", toJSON(authresp))

	bw := bytes.NewBuffer(make([]byte, 0, 1024))
	err := cbor.WriteTo(bw, authresp)
	if err != nil {
		log.Fatalf("cannt encode signin req: %v", err)
	}

	resp, err := co.Post(context.Background(), session.ResourceURI, message.AppCBOR, bytes.NewReader(bw.Bytes()))
	if err != nil {
		log.Fatalf("error sending request to signin: %v", err)
	}
	if resp.Code() != codes.Changed {
		log.Fatalf("error get coap code for sigin: %v", resp.Code())
	}
}

func toJSON(v interface{}) string {
	d, err := json.Encode(v)
	if err != nil {
		log.Fatalf("cannot decode rd resp: %v", err)
	}
	return string(d)
}

func decodePayload(resp *pool.Message) {
	mt, err := resp.Options().ContentFormat()

	buf := fmt.Sprint("-------------------COAP-RESPONSE------------------\n",
		"Code: ", resp.Code(), "\n",
		"ContentFormat: ", mt, "\n",
		"Payload: ",
	)
	if err == nil {
		bufr, err := io.ReadAll(resp.Body())
		if err != nil {
			buf += fmt.Sprintf("cannot read body: %v", err)
			log.Print(buf)
			return
		}
		switch mt {
		case message.AppCBOR, message.AppOcfCbor:
			s, err := cbor.ToJSON(bufr)
			if err != nil {
				buf += fmt.Sprintf("Cannot encode %v to JSON: %v", bufr, err)
			} else {
				buf += fmt.Sprintf("%v\n", s)
			}
		case message.TextPlain:

			buf += fmt.Sprintf("%v\n", string(bufr))
		case message.AppJSON:
			buf += fmt.Sprintf("%v\n", string(bufr))
		case message.AppXML:
			buf += fmt.Sprintf("%v\n", string(bufr))
		default:
			buf += fmt.Sprintf("%v\n", bufr)
		}
	}
	log.Print(buf)
}

func Conn(addr string) (*client.Conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse url %v: %w", addr, err)
	}
	address, err := net.ParseURL(u)
	if err != nil {
		return nil, fmt.Errorf("cannot parse address %v: %w", addr, err)
	}

	dialError := func(err error) error {
		return fmt.Errorf("error dialing: %w", err)
	}
	switch address.GetScheme() {
	case "coap+tcp":
		co, err := coap.Dial(address.String(), options.WithMaxMessageSize(256*1024))
		if err != nil {
			return nil, dialError(err)
		}
		return co, nil
	case "coaps+tcp":
		co, err := coap.Dial(address.String(), options.WithTLS(&tls.Config{
			InsecureSkipVerify: true,
		}), options.WithMaxMessageSize(256*1024))
		if err != nil {
			return nil, dialError(err)
		}
		return co, nil
	default:
		return nil, fmt.Errorf("invalid scheme %v of address %v", address.GetScheme(), address)
	}
}

func deleteResource(co *client.Conn, href string) {
	deleteError := func(err error) {
		log.Fatalf("cannot delete resource: %v", err)
	}
	resp, err := co.Delete(context.Background(), href)
	if err != nil {
		deleteError(err)
	}
	decodePayload(resp)
}

func updateResource(co *client.Conn, href string, contentFormat int) {
	updateError := func(err error) {
		log.Fatalf("cannot update resource: %v", err)
	}
	b := bytes.NewBuffer(make([]byte, 0, 124))
	_, err := b.ReadFrom(os.Stdin)
	if err != nil {
		updateError(err)
	}
	resp, err := co.Post(context.Background(), href, message.MediaType(contentFormat), bytes.NewReader(b.Bytes())) //nolint:gosec
	if err != nil {
		updateError(err)
	}
	decodePayload(resp)
}

func createResource(co *client.Conn, href string, contentFormat int) {
	createError := func(err error) {
		log.Fatalf("cannot create resource: %v", err)
	}
	b := bytes.NewBuffer(make([]byte, 0, 124))
	_, err := b.ReadFrom(os.Stdin)
	if err != nil {
		createError(err)
	}
	req, err := co.NewPostRequest(context.Background(), href, message.MediaType(contentFormat), os.Stdin) //nolint:gosec
	if err != nil {
		createError(err)
	}
	req.SetOptionString(message.URIQuery, uri.InterfaceQueryKeyPrefix+interfaces.OC_IF_CREATE)
	resp, err := co.Do(req)
	if err != nil {
		createError(err)
	}
	decodePayload(resp)
}

func observerResource(co *client.Conn, href string) {
	obs, err := co.Observe(context.Background(), href, func(req *pool.Message) {
		decodePayload(req)
	})
	if err != nil {
		log.Fatalf("cannot observe resource: %v", err)
	}
	defer func() {
		err := obs.Cancel(context.Background())
		if err != nil {
			fmt.Printf("failed to cancel observation: %v", err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	fmt.Println("exiting")
}

func getResource(co *client.Conn, href, resIf, resRt string) {
	var opts message.Options
	if resIf != "" {
		v := uri.InterfaceQueryKeyPrefix + resIf
		buf := make([]byte, len(v))
		opts, _, _ = opts.AddString(buf, message.URIQuery, v)
	}
	if resRt != "" {
		v := "rt=" + resRt
		buf := make([]byte, len(v))
		opts, _, _ = opts.AddString(buf, message.URIQuery, v)
	}
	resp, err := co.Get(context.Background(), href, opts...)
	if err != nil {
		log.Fatalf("cannot get resource: %v", err)
	}
	decodePayload(resp)
}

func main() {
	addr := flag.String("cis", "coap+tcp://127.0.0.1:5683", "address")
	authCode := flag.String("signUp", "", "authcode")
	accesstoken := flag.String("signIn", "", "accesstoken")
	di := flag.String("di", "testUtility", "device id")
	uid := flag.String("uid", "", "user id")
	href := flag.String("href", resources.ResourceURI, "href")
	resIf := flag.String("if", "", "interface")
	get := flag.Bool("get", true, "get resource(default)")
	resRt := flag.String("rt", "", "resource type")
	observe := flag.Bool("observe", false, "observe resource")
	update := flag.Bool("update", false, "update resource, content is expected in stdin")
	resDelete := flag.Bool("delete", false, "delete resource")
	create := flag.Bool("create", false, "create resource, content is expected in stdin")
	contentFormat := flag.Int("contentFormat", int(message.AppJSON), "contentFormat for update resource")

	flag.Parse()

	co, err := Conn(*addr)
	if err != nil {
		log.Fatal(err)
	}

	if *authCode != "" {
		*accesstoken, *di = signUpWithAuthCode(co, *authCode, *di)
	}
	if *accesstoken != "" && *uid != "" {
		signInReq := authResp{
			Accesstoken: *accesstoken,
			UID:         *uid,
			DeviceID:    *di,
			Login:       true,
		}
		signIn(co, signInReq)
	}

	switch {
	case *resDelete:
		deleteResource(co, *href)
	case *update:
		updateResource(co, *href, *contentFormat)
	case *create:
		createResource(co, *href, *contentFormat)
	case *observe:
		observerResource(co, *href)
	case *get:
		getResource(co, *href, *resIf, *resRt)
	default:
		log.Fatal("unknown command")
	}
}
