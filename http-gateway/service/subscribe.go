package service

import (
	"net/http"
	"sync"
	"time"

	"github.com/plgd-dev/kit/log"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

func NewObservationManager() (*ObservationManager, error) {
	ws := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	m := ObservationManager{
		ws:           ws,
		observations: make(map[string]map[string]SubscribeSession),
	}
	return &m, nil
}

type ObservationManager struct {
	ws           websocket.Upgrader
	lock         sync.Mutex
	observations map[string]map[string]SubscribeSession
}

type ObservationResolver interface {
	StartObservation(r *http.Request, ws *websocket.Conn) (SubscribeSession, error)
	StopObservation(subscriptionID string) error
}

func NewSubscriptionSession(ws *websocket.Conn) subscribeSession {
	ob := subscribeSession{
		clientId: GetClientID(ws),
		ws:       ws,
		writeCh:  make(chan interface{}, 10),
	}
	return ob
}

type subscribeSession struct {
	clientId           string
	subscriptionId     string
	ws                 *websocket.Conn
	writeCh            chan interface{}
	closedWriteChMutex sync.Mutex
	closedWriteCh      bool
}

type SubscribeSession interface {
	ClientId() string
	SubscriptionId() string
	SetSubscriptionId(id string)
	Write(data interface{})
	ReadLoop(rh *RequestHandler, ob ObservationResolver)
	WriteLoop()
	OnClose()
}

func (s *subscribeSession) ClientId() string {
	return s.clientId
}

func (s *subscribeSession) SubscriptionId() string {
	return s.subscriptionId
}

func (s *subscribeSession) SetSubscriptionId(id string) {
	s.subscriptionId = id
}

func (s *subscribeSession) Write(data interface{}) {
	s.closedWriteChMutex.Lock()
	defer s.closedWriteChMutex.Unlock()
	if s.closedWriteCh {
		return
	}
	s.writeCh <- data
}

func (s *subscribeSession) OnClose() {
	s.closedWriteChMutex.Lock()
	defer s.closedWriteChMutex.Unlock()
	if s.closedWriteCh {
		return
	}
	s.closedWriteCh = true
	close(s.writeCh)
}

func (s *subscribeSession) Error(err error) {
	log.Errorf("%w", err)
	s.ws.Close()
}

func (requestHandler *RequestHandler) ServeWs(w http.ResponseWriter, r *http.Request, ob ObservationResolver) error {
	c, err := requestHandler.manager.ws.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("unable to upgrade into websocket: %w", err)
		return err
	}
	s, err := ob.StartObservation(r, c)
	if err != nil {
		c.Close()
		log.Error("unable to start observation: %w", err)
		return err
	}
	requestHandler.addSession(s)
	go s.ReadLoop(requestHandler, ob)
	go s.WriteLoop()
	return nil
}

func (s *subscribeSession) ReadLoop(rh *RequestHandler, ob ObservationResolver) {
	defer func() {
		err := ob.StopObservation(s.subscriptionId)
		if err != nil {
			log.Errorf("unable to close observation: %w", err)
		}
		rh.removeSession(s)
	}()
	s.ws.SetReadDeadline(time.Now().Add(pongWait))
	s.ws.SetPongHandler(func(string) error {
		s.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := s.ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *subscribeSession) WriteLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		s.ws.Close()
	}()
	for {
		select {
		case message, ok := <-s.writeCh:
			if !ok {
				// in this case, cloud subscription is already closed
				err := s.ws.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					log.Error("unable to send websocket close message. ws is already closed")
				}
				return
			}
			var err error
			if bytes, ok := message.([]byte); ok {
				err = s.ws.WriteMessage(websocket.TextMessage, bytes)
			} else {
				err = s.ws.WriteJSON(message)
			}
			if err != nil {
				//when this routine end. Ping message will not be send and ws will be closed
				return
			}
		case <-ticker.C:
			s.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (requestHandler *RequestHandler) addSession(s SubscribeSession) {
	requestHandler.manager.lock.Lock()
	defer requestHandler.manager.lock.Unlock()
	if c, ok := requestHandler.manager.observations[s.ClientId()]; ok {
		c[s.SubscriptionId()] = s
	} else {
		requestHandler.manager.observations[s.ClientId()] = make(map[string]SubscribeSession)
		requestHandler.manager.observations[s.ClientId()][s.SubscriptionId()] = s
	}
}

func (requestHandler *RequestHandler) removeSession(s SubscribeSession) {
	requestHandler.manager.lock.Lock()
	defer requestHandler.manager.lock.Unlock()
	if c, ok := requestHandler.manager.observations[s.ClientId()]; ok {
		if _, exists := c[s.SubscriptionId()]; exists {
			delete(c, s.SubscriptionId())
			if len(c) == 0 {
				delete(c, s.ClientId())
			}
		}
	}
}
func (requestHandler *RequestHandler) pop() map[string]map[string]SubscribeSession {
	requestHandler.manager.lock.Lock()
	defer requestHandler.manager.lock.Unlock()
	observations := requestHandler.manager.observations
	requestHandler.manager.observations = make(map[string]map[string]SubscribeSession)
	return observations
}

func GetClientID(ws *websocket.Conn) string {
	return ws.RemoteAddr().String()
}
