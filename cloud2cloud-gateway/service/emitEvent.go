package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	netHttp "net/http"
	"strconv"
	"time"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/http/transport"
)

type incrementSubscriptionSequenceNumberFunc func(ctx context.Context) (uint64, error)

func emitEvent(ctx context.Context, eventType events.EventType, s store.Subscription, incrementSubscriptionSequenceNumber incrementSubscriptionSequenceNumberFunc, rep interface{}) (remove bool, err error) {
	log.Debugf("emitEvent: %v: %+v", eventType, s)
	defer log.Debugf("emitEvent done: %v: %+v", eventType, s)

	trans := transport.NewDefaultTransport()
	trans.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	client := netHttp.Client{
		Transport: trans,
	}
	encoder, err := getEncoder(s.ContentType)
	if err != nil {
		return false, fmt.Errorf("cannot get encoder: %w", err)
	}
	seqNum, err := incrementSubscriptionSequenceNumber(ctx)
	if err != nil {
		return false, fmt.Errorf("cannot increment sequence number: %w", err)
	}

	r, w := io.Pipe()

	req, err := netHttp.NewRequest("POST", s.URL, r)
	if err != nil {
		return false, fmt.Errorf("cannot create post request: %w", err)
	}
	timestamp := time.Now()
	req.Header.Set(events.EventTypeKey, string(eventType))
	req.Header.Set(events.SubscriptionIDKey, s.ID)
	req.Header.Set(events.SequenceNumberKey, strconv.FormatUint(seqNum, 10))
	req.Header.Set(events.CorrelationIDKey, s.CorrelationID)
	req.Header.Set(events.EventTimestampKey, strconv.FormatInt(timestamp.Unix(), 10))
	var body []byte
	if rep != nil {
		body, err = encoder(rep)
		if err != nil {
			return false, fmt.Errorf("cannot encode data to body: %w", err)
		}
		req.Header.Set(events.ContentTypeKey, s.ContentType)
		go func() {
			defer w.Close()
			if len(body) > 0 {
				_, err := w.Write(body)
				if err != nil {
					log.Errorf("cannot write data to client: %v", err)
				}
			}
		}()
	} else {
		w.Close()
	}
	req.Header.Set(events.EventSignatureKey, events.CalculateEventSignature(
		s.SigningSecret,
		req.Header.Get(events.ContentTypeKey),
		eventType,
		req.Header.Get(events.SubscriptionIDKey),
		seqNum,
		timestamp,
		body,
	))
	req.Header.Set("Connection", "close")
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("cannot post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != netHttp.StatusOK {
		errBody, _ := ioutil.ReadAll(resp.Body)
		return resp.StatusCode == netHttp.StatusGone, fmt.Errorf("%v: unexpected statusCode %v: body: '%v'", s.URL, resp.StatusCode, string(errBody))
	}
	return eventType == events.EventType_SubscriptionCanceled, nil
}
