package service

import (
	"sync"
	"time"

	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwTestService "github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/test/coap-gateway/test"
)

const (
	SignUpKey       = "SignUp"  // register
	SignOffKey      = "SignOff" // deregister
	SignInKey       = "SignIn"  // log in
	SignOutKey      = "SignOut" // log out
	PublishKey      = "Publish"
	UnpublishKey    = "Unpublish"
	RefreshTokenKey = "RefreshToken"
)

type CoapHandlerWithCounter struct {
	*coapgwTest.DefaultObserverHandler

	CallCounter struct {
		Data map[string]int
		Lock sync.Mutex
	}

	signedInChan  chan int
	signedOffChan chan int
	refreshChan   chan int
}

func NewCoapHandlerWithCounter(atLifetime int64) *CoapHandlerWithCounter {
	dh := coapgwTest.MakeDefaultObserverHandler(atLifetime)
	return &CoapHandlerWithCounter{
		DefaultObserverHandler: &dh,
		CallCounter: struct {
			Data map[string]int
			Lock sync.Mutex
		}{
			Data: make(map[string]int),
		},

		signedInChan:  make(chan int, 256),
		signedOffChan: make(chan int, 256),
		refreshChan:   make(chan int, 256),
	}
}

func (ch *CoapHandlerWithCounter) SignUp(req coapgwService.CoapSignUpRequest) (coapgwService.CoapSignUpResponse, error) {
	resp, err := ch.DefaultObserverHandler.SignUp(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignUpKey]++
	ch.CallCounter.Lock.Unlock()
	return resp, err
}

func (ch *CoapHandlerWithCounter) SignOff() error {
	err := ch.DefaultObserverHandler.SignOff()
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignOffKey]++
	signOffCount, ok := ch.CallCounter.Data[SignOffKey]
	if ok {
		sendToChan(ch.signedOffChan, signOffCount)
	}
	ch.CallCounter.Lock.Unlock()
	return err
}

func sendToChan(c chan int, v int) {
	select {
	case c <- v:
	default:
	}
}

func resetChan(c chan int) {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}

func waitForAction(c chan int, timeout time.Duration) int {
	select {
	case v := <-c:
		return v
	case <-time.After(timeout):
		return -1
	}
}

func (ch *CoapHandlerWithCounter) ResetSignOff() {
	resetChan(ch.signedOffChan)
}

func (ch *CoapHandlerWithCounter) WaitForSignOff(timeout time.Duration) bool {
	return waitForAction(ch.signedOffChan, timeout) >= 0
}

func (ch *CoapHandlerWithCounter) WaitForFirstSignOff(timeout time.Duration) bool {
	return waitForAction(ch.signedOffChan, timeout) == 1
}

func (ch *CoapHandlerWithCounter) SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error) {
	resp, err := ch.DefaultObserverHandler.SignIn(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignInKey]++
	signInCount, ok := ch.CallCounter.Data[SignInKey]
	if ok {
		sendToChan(ch.signedInChan, signInCount)
	}
	ch.CallCounter.Lock.Unlock()
	return resp, err
}

func (ch *CoapHandlerWithCounter) ResetSignIn() {
	resetChan(ch.signedInChan)
}

func (ch *CoapHandlerWithCounter) WaitForSignIn(timeout time.Duration) bool {
	return waitForAction(ch.signedInChan, timeout) >= 0
}

func (ch *CoapHandlerWithCounter) WaitForFirstSignIn(timeout time.Duration) bool {
	return waitForAction(ch.signedInChan, timeout) == 1
}

func (ch *CoapHandlerWithCounter) SignOut(req coapgwService.CoapSignInReq) error {
	err := ch.DefaultObserverHandler.SignOut(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignOutKey]++
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *CoapHandlerWithCounter) PublishResources(req coapgwTestService.PublishRequest) error {
	err := ch.DefaultObserverHandler.PublishResources(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[PublishKey]++
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *CoapHandlerWithCounter) UnpublishResources(req coapgwTestService.UnpublishRequest) error {
	err := ch.DefaultObserverHandler.UnpublishResources(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[UnpublishKey]++
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *CoapHandlerWithCounter) RefreshToken(req coapgwService.CoapRefreshTokenReq) (coapgwService.CoapRefreshTokenResp, error) {
	resp, err := ch.DefaultObserverHandler.RefreshToken(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[RefreshTokenKey]++
	refreshCount, ok := ch.CallCounter.Data[RefreshTokenKey]
	if ok {
		sendToChan(ch.refreshChan, refreshCount)
	}
	ch.CallCounter.Lock.Unlock()
	return resp, err
}

func (ch *CoapHandlerWithCounter) ResetRefreshToken() {
	resetChan(ch.refreshChan)
}

func (ch *CoapHandlerWithCounter) WaitForRefreshToken(timeout time.Duration) bool {
	return waitForAction(ch.refreshChan, timeout) >= 0
}

func (ch *CoapHandlerWithCounter) WaitForFirstRefreshToken(timeout time.Duration) bool {
	return waitForAction(ch.refreshChan, timeout) == 1
}
