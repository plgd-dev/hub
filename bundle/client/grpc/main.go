package main

import (
	"bytes"
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

	pbGW "github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/json"
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

func getServiceToken(authAddr string, tls *tls.Config) (string, error) {
	reqBody := map[string]string{
		"grant_type":    string(service.AllowedGrantType_CLIENT_CREDENTIALS),
		uri.ClientIDKey: service.ClientTest,
		uri.AudienceKey: "test",
	}
	d, err := json.Encode(reqBody)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest(http.MethodPost, "https://"+authAddr+"/oauth/token", bytes.NewReader(d))
	if err != nil {
		return "", err
	}
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tls,
		},
	}
	res, err := c.Do(request)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("returns statu code %v", res.StatusCode)
	}
	var body map[string]string
	err = json.ReadFrom(res.Body, &body)
	if err != nil {
		return "", err
	}
	token := body["access_token"]
	if token == "" {
		return "", fmt.Errorf("token not found in body")
	}
	return token, nil
}

func main() {
	addr := flag.String("addr", "localhost:443", "address")
	accesstoken := flag.String("accesstoken", "", "accesstoken")
	authAddr := flag.String("authaddr", "localhost:443", "authorization service address")
	deviceID := flag.String("deviceid", "", "deviceID")
	href := flag.String("href", "", "href")
	get := flag.Bool("get", true, "get resources(default) filtered by deviceid and href")
	getDevices := flag.Bool("getdevices", false, "get devices")
	//observe := flag.Bool("observe", false, "observe resource")
	update := flag.Bool("update", false, "update resource, content is expected in stdin")
	delete := flag.Bool("delete", false, "delete resource")
	create := flag.Bool("create", false, "create resource, content is expected in stdin")

	contentFormat := flag.Int("contentFormat", int(message.AppJSON), "contentFormat for update resource")

	flag.Parse()

	tlsCfg := tls.Config{
		InsecureSkipVerify: true,
	}
	var err error
	if *accesstoken == "" {
		*accesstoken, err = getServiceToken(*authAddr, &tlsCfg)
		if err != nil {
			log.Fatalf("cannot get access token: %v", err)
		}
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(credentials.NewTLS(&tlsCfg)))
	if err != nil {
		log.Fatalf("Error dialing: %v", err)
	}

	ocfGW := pbGW.NewGrpcGatewayClient(conn)
	//ocfGWHelper := cloud.NewClient(ocfGW)
	ctx := kitNetGrpc.CtxWithToken(context.Background(), *accesstoken)
	switch {
	case *delete:
		resp, err := ocfGW.DeleteResource(ctx, &pbGW.DeleteResourceRequest{
			ResourceId: &commands.ResourceId{
				DeviceId: *deviceID,
				Href:     *href,
			},
		})
		if err != nil {
			log.Fatalf("cannot delete resource: %v", err)
		}
		d, err := json.Encode(resp)
		if err != nil {
			log.Fatalf("cannot encode resp to json: %v", err)
		}
		fmt.Println(string(d))
	case *update:
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("cannot read data for update resource: %v", err)
		}
		resp, err := ocfGW.UpdateResource(ctx, &pbGW.UpdateResourceRequest{
			ResourceId: &commands.ResourceId{
				DeviceId: *deviceID,
				Href:     *href,
			},
			Content: &pbGW.Content{
				ContentType: message.MediaType(*contentFormat).String(),
				Data:        data,
			},
		})
		if err != nil {
			log.Fatalf("cannot update resource: %v", err)
		}
		d, err := json.Encode(resp)
		if err != nil {
			log.Fatalf("cannot encode resp to json: %v", err)
		}
		fmt.Println(string(d))
	case *create:
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("cannot read data for create resource: %v", err)
		}
		resp, err := ocfGW.CreateResource(ctx, &pbGW.CreateResourceRequest{
			ResourceId: &commands.ResourceId{
				DeviceId: *deviceID,
				Href:     *href,
			},
			Content: &pbGW.Content{
				ContentType: message.MediaType(*contentFormat).String(),
				Data:        data,
			},
		})
		if err != nil {
			log.Fatalf("cannot create resource: %v", err)
		}
		d, err := json.Encode(resp)
		if err != nil {
			log.Fatalf("cannot encode resp to json: %v", err)
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
			log.Fatalf("cannot encode resp to json: %v", err)
		}
		fmt.Println(string(d))
	case *get:
		var deviceIdsFilter []string
		if *deviceID != "" {
			deviceIdsFilter = append(deviceIdsFilter, *deviceID)
		}
		var resourceIdsFilter []*commands.ResourceId
		if *href != "" {
			resourceIdsFilter = append(resourceIdsFilter, &commands.ResourceId{
				DeviceId: *deviceID,
				Href:     *href,
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
			log.Fatalf("cannot encode resp to json: %v", err)
		}
		fmt.Println(string(d))
	default:
		if err != nil {
			log.Fatal("unknown command")
		}
	}
}
