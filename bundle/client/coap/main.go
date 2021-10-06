package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/plgd-dev/go-coap/v2/message"
	codes "github.com/plgd-dev/go-coap/v2/message/codes"
	coap "github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/pkg/ocf"
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

func signUp(co *coap.ClientConn, authreq authReq) authResp {
	bw := bytes.NewBuffer(make([]byte, 0, 1024))
	err := cbor.WriteTo(bw, authreq)
	if err != nil {
		log.Fatalf("cannt encode signup req: %v", err)
	}

	resp, err := co.Post(context.Background(), "/oic/sec/account", message.AppCBOR, bytes.NewReader(bw.Bytes()))
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

func signIn(co *coap.ClientConn, authresp authResp) {
	log.Printf("authresp: \n%v\n", toJSON(authresp))

	bw := bytes.NewBuffer(make([]byte, 0, 1024))
	err := cbor.WriteTo(bw, authresp)
	if err != nil {
		log.Fatalf("cannt encode signin req: %v", err)
	}

	resp, err := co.Post(context.Background(), "/oic/sec/session", message.AppCBOR, bytes.NewReader(bw.Bytes()))
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
		bufr, err := ioutil.ReadAll(resp.Body())
		if err != nil {
			buf = buf + fmt.Sprintf("cannot read body: %v", err)
			log.Print(buf)
			return
		}
		switch mt {
		case message.AppCBOR, message.AppOcfCbor:
			s, err := cbor.ToJSON(bufr)
			if err != nil {
				buf = buf + fmt.Sprintf("Cannot encode %v to JSON: %v", bufr, err)
			} else {
				buf = buf + fmt.Sprintf("%v\n", s)
			}
		case message.TextPlain:

			buf = buf + fmt.Sprintf("%v\n", string(bufr))
		case message.AppJSON:
			buf = buf + fmt.Sprintf("%v\n", string(bufr))
		case message.AppXML:
			buf = buf + fmt.Sprintf("%v\n", string(bufr))
		default:
			buf = buf + fmt.Sprintf("%v\n", bufr)
		}
	}
	log.Print(buf)
}

func main() {
	addr := flag.String("cis", "coap+tcp://127.0.0.1:5683", "address")
	authCode := flag.String("signUp", "", "authcode")
	accesstoken := flag.String("signIn", "", "accesstoken")
	di := flag.String("di", "testUtility", "device id")
	uid := flag.String("uid", "", "user id")
	href := flag.String("href", "/oic/res", "href")
	resIf := flag.String("if", "", "interface")
	get := flag.Bool("get", true, "get resource(default)")
	discoverRt := flag.String("rt", "", "resource type")
	observe := flag.Bool("observe", false, "observe resource")
	update := flag.Bool("update", false, "update resource, content is expected in stdin")
	delete := flag.Bool("delete", false, "delete resource")
	create := flag.Bool("create", false, "create resource, content is expected in stdin")

	contentFormat := flag.Int("contentFormat", int(message.AppJSON), "contentFormat for update resource")

	flag.Parse()

	u, err := url.Parse(*addr)
	if err != nil {
		log.Fatalf("cannot parse url %v: %v", *addr, err)
	}
	address, err := net.ParseURL(u)
	if err != nil {
		log.Fatalf("cannot parse address %v: %v", *addr, err)
	}

	var co *coap.ClientConn
	switch address.GetScheme() {
	case "coap+tcp":
		co, err = coap.Dial(address.String(), coap.WithMaxMessageSize(256*1024))
		if err != nil {
			log.Fatalf("Error dialing: %v", err)
		}
	case "coaps+tcp":
		co, err = coap.Dial(address.String(), coap.WithTLS(&tls.Config{
			InsecureSkipVerify: true,
		}), coap.WithMaxMessageSize(256*1024))
		if err != nil {
			log.Fatalf("Error dialing: %v", err)
		}
	default:
		log.Fatalf("invalid scheme %v of address %v", address.GetScheme(), address)
	}

	if err != nil {
		log.Fatalf("Error dialing: %v", err)
	}

	if *authCode != "" {
		authreq := authReq{
			Accesstoken:  *authCode,
			DeviceID:     *di,
			AuthProvider: "plgd",
		}
		authresp := signUp(co, authreq)
		*accesstoken = authresp.Accesstoken
		*uid = authresp.UID
		authresp.DeviceID = *di
		authresp.Login = true
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
	case *delete:
		resp, err := co.Delete(context.Background(), *href)
		if err != nil {
			log.Fatalf("cannot delete resource: %v", err)
		}
		decodePayload(resp)
	case *update:
		b := bytes.NewBuffer(make([]byte, 0, 124))
		_, err := b.ReadFrom(os.Stdin)
		if err != nil {
			log.Fatalf("cannot update resource: %v", err)
		}
		resp, err := co.Post(context.Background(), *href, message.MediaType(*contentFormat), bytes.NewReader(b.Bytes()))
		if err != nil {
			log.Fatalf("cannot update resource: %v", err)
		}
		decodePayload(resp)
	case *create:
		b := bytes.NewBuffer(make([]byte, 0, 124))
		_, err := b.ReadFrom(os.Stdin)
		if err != nil {
			log.Fatalf("cannot update resource: %v", err)
		}
		req, err := coap.NewPostRequest(context.Background(), *href, message.MediaType(*contentFormat), os.Stdin)
		if err != nil {
			log.Fatalf("cannot create resource: %v", err)
		}
		req.SetOptionString(message.URIQuery, "if="+ocf.OC_IF_CREATE)
		resp, err := co.Do(req)
		if err != nil {
			log.Fatalf("cannot update resource: %v", err)
		}
		decodePayload(resp)
	case *observe:
		obs, err := co.Observe(context.Background(), *href, func(req *pool.Message) {
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
	case *get:
		var opts message.Options
		if *resIf != "" {
			v := "if=" + *resIf
			buf := make([]byte, len(v))
			opts, _, _ = opts.AddString(buf, message.URIQuery, v)
		}
		if *discoverRt != "" {
			v := "rt=" + *discoverRt
			buf := make([]byte, len(v))
			opts, _, _ = opts.AddString(buf, message.URIQuery, v)
		}
		resp, err := co.Get(context.Background(), *href, opts...)
		if err != nil {
			log.Fatalf("cannot get resource: %v", err)
		}
		decodePayload(resp)
	default:
		if err != nil {
			log.Fatal("unknown command")
		}
	}
}
