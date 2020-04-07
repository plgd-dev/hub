package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	coap "github.com/go-ocf/go-coap"
	pbGW "github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/codec/json"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/net/http/transport"
)

func toJSON(v interface{}) string {
	d, err := json.Encode(v)
	if err != nil {
		log.Fatalf("cannot decode rd resp: %v", err)
	}
	return string(d)
}

func decodePayload(resp *pbGW.Content) {
	/*
		buf := fmt.Sprint("-------------------COAP-RESPONSE------------------\n",
			"Code: ", resp.Code(), "\n",
			"ContentFormat: ", resp.Options(coap.ContentFormat), "\n",
			"Payload: ",
		)
		if mediaType, ok := resp.Option(coap.ContentFormat).(coap.MediaType); ok {
			switch mediaType {
			case coap.AppCBOR, coap.AppOcfCbor:
				var m interface{}
				err := codec.NewDecoderBytes(resp.Payload(), new(codec.CborHandle)).Decode(&m)
				bw := new(bytes.Buffer)
				h := new(codec.JsonHandle)
				h.BasicHandle.Canonical = true
				err = codec.NewEncoder(bw, h).Encode(m)
				if err != nil {
					buf = buf + fmt.Sprintf("Cannot encode %v to JSON: %v", m, err)
				} else {
					buf = buf + fmt.Sprintf("%v\n", bw.String())
				}
			case coap.TextPlain:
				buf = buf + fmt.Sprintf("%v\n", string(resp.Payload()))
			case coap.AppJSON:
				buf = buf + fmt.Sprintf("%v\n", string(resp.Payload()))
			case coap.AppXML:
				buf = buf + fmt.Sprintf("%v\n", string(resp.Payload()))
			default:
				buf = buf + fmt.Sprintf("%v\n", resp.Payload())
			}
		} else {
			buf = buf + fmt.Sprintf("%v\n", resp.Payload())
		}
		log.Printf(buf)
	*/
}

func main() {
	addr := flag.String("addr", "localhost:9084", "address")
	accesstoken := flag.String("accesstoken", "", "accesstoken")
	authAddr := flag.String("authaddr", "localhost:9085", "authorization serivce address")
	deviceID := flag.String("deviceid", "", "deviceID")
	href := flag.String("href", "", "href")
	get := flag.Bool("get", true, "get resources(default) filtered by deviceid and href")
	getDevices := flag.Bool("getdevices", false, "get devices")
	//observe := flag.Bool("observe", false, "observe resource")
	update := flag.Bool("update", false, "update resource, content is expceted in stdin")

	contentFormat := flag.Int("contentFormat", int(coap.AppJSON), "contentFormat for update resource")

	flag.Parse()

	tlsCfg := tls.Config{
		InsecureSkipVerify: true,
	}
	if *accesstoken == "" {
		t := transport.NewDefaultTransport()
		t.TLSClientConfig = &tlsCfg
		c := http.Client{
			Transport: t,
		}
		resp, err := c.Get("https://" + *authAddr + "/api/authz/token")
		if err != nil {
			log.Fatalf("cannot get access token: %v", err)
		}
		defer resp.Body.Close()
		type at struct {
			AccessToken string `json:"access_token"`
		}
		var a at
		err = json.ReadFrom(resp.Body, &a)
		if err != nil {
			log.Fatalf("cannot read access token: %v", err)
		}
		*accesstoken = a.AccessToken
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(credentials.NewTLS(&tlsCfg)))
	if err != nil {
		log.Fatalf("Error dialing: %v", err)
	}

	ocfGW := pbGW.NewGrpcGatewayClient(conn)
	//ocfGWHelper := cloud.NewClient(ocfGW)
	ctx := kitNetGrpc.CtxWithToken(context.Background(), *accesstoken)
	switch {
	case *update:
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("cannot read data for update value: %v", err)
		}
		resp, err := ocfGW.UpdateResourcesValues(ctx, &pbGW.UpdateResourceValuesRequest{
			ResourceId: &pbGW.ResourceId{
				DeviceId:         *deviceID,
				ResourceLinkHref: *href,
			},
			Content: &pbGW.Content{
				ContentType: coap.MediaType(*contentFormat).String(),
				Data:        data,
			},
		})
		if err != nil {
			log.Fatalf("cannot update value: %v", err)
		}
		d, err := json.Encode(resp)
		if err != nil {
			log.Fatalf("cannot encode device: %v", err)
		}
		fmt.Println(string(d))
	/*
		case *observe:
			log.Fatalf("not implemented")

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs
			fmt.Println("exiting")
	*/
	case *getDevices:
		getClient, err := ocfGW.GetDevices(ctx, &pbGW.GetDevicesRequest{})
		if err != nil {
			log.Fatalf("cannot get devices: %v", err)
		}
		devices := make([]*pbGW.Device, 0, 4)
		for {
			resp, err := getClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("cannot recv device: %v", err)
			}
			devices = append(devices, resp)
		}
		d, err := json.Encode(devices)
		if err != nil {
			log.Fatalf("cannot encode device: %v", err)
		}
		fmt.Println(string(d))
	case *get:
		var deviceIdsFilter []string
		if *deviceID != "" {
			deviceIdsFilter = append(deviceIdsFilter, *deviceID)
		}
		var resourceIdsFilter []*pbGW.ResourceId
		if *href != "" {
			resourceIdsFilter = append(resourceIdsFilter, &pbGW.ResourceId{
				DeviceId:         *deviceID,
				ResourceLinkHref: *href,
			})
		}
		getClient, err := ocfGW.RetrieveResourcesValues(ctx, &pbGW.RetrieveResourcesValuesRequest{
			ResourceIdsFilter: resourceIdsFilter,
			DeviceIdsFilter:   deviceIdsFilter,
		})
		if err != nil {
			log.Fatalf("cannot retrieve values: %v", err)
		}
		resources := make([]*pbGW.ResourceValue, 0, 4)
		for {
			resp, err := getClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("cannot recv value: %v", err)
			}
			resources = append(resources, resp)
		}
		d, err := json.Encode(resources)
		if err != nil {
			log.Fatalf("cannot encode device: %v", err)
		}
		fmt.Println(string(d))
	default:
		if err != nil {
			log.Fatal("unknown command")
		}
	}
}
