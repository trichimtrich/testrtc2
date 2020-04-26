package network

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"testrtc2/screen"

	"github.com/gorilla/websocket"
)

type WebSocket struct {
	screen *screen.Screen
	rtc    *WebRTC
	conn   *websocket.Conn
	userID string
	peerID string
	mu     sync.Mutex // write mutex
}

type ActionPacket struct {
	Action string `json:"action"`
}

type InitPacket struct {
	ActionPacket
	ID string `json:"id"`
}

type Message struct {
	Topic string `json:"topic"`
	Body  string `json:"body"`
}

type ErrorPacket struct {
	ActionPacket
	Msg string `json:"msg"`
}

type RecvPacket struct {
	ActionPacket
	From string  `json:"from"`
	Msg  Message `json:"msg"`
}

type SendPacket struct {
	ActionPacket
	To  string  `json:"to"`
	Msg Message `json:"msg"`
}

func NewWebSocket(screen *screen.Screen) *WebSocket {
	return &WebSocket{screen, nil, nil, "", "", sync.Mutex{}}
}

func (ws *WebSocket) SetWebRTC(rtc *WebRTC) {
	ws.rtc = rtc
}

func (ws *WebSocket) Reset() {
	ws.userID = ""
	ws.peerID = ""

	if ws.conn != nil {
		ws.conn.Close()
		ws.screen.Log("[System] Close previous web socket")
		ws.conn = nil
	}
}

func (ws *WebSocket) GetPeer() string {
	return ws.peerID
}

func (ws *WebSocket) SetPeer(id string) {
	ws.screen.Log("[System] Set Peer ID: " + id)
	ws.peerID = id
}

func (ws *WebSocket) sendSafePacket(data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.conn.WriteMessage(websocket.TextMessage, data)
}

func (ws *WebSocket) SendMessage(msg Message) {
	sp := SendPacket{ActionPacket{"send"}, ws.peerID, msg}
	packet, err := json.Marshal(sp)
	if err != nil {
		ws.screen.Log("[WebSocket] json encode packet failed: " + err.Error())
		return
	}
	err = ws.sendSafePacket(packet)
	if err != nil {
		ws.screen.Log("[WebSocket] write message failed: " + err.Error())
	}

	ws.screen.Log(fmt.Sprintf("[WebSocket] sent mail to %s: <%s> %d bytes", ws.peerID, msg.Topic, len(msg.Body)))
}

func (ws *WebSocket) handleMessage(message []byte) (err error) {
	var ap ActionPacket
	err = json.Unmarshal(message, &ap)
	if err != nil {
		return
	}
	switch ap.Action {
	case "init": // identity
		var ip InitPacket
		err = json.Unmarshal(message, &ip)
		if err != nil {
			return
		}
		ws.userID = ip.ID
		ws.screen.Log("[WebSocket] my new ID: " + ip.ID)
		ws.screen.SetTitle(fmt.Sprintf("My ID = %s. Enter command ...", ip.ID))

	case "error": // error
		var ep ErrorPacket
		err = json.Unmarshal(message, &ep)
		if err != nil {
			return
		}
		ws.screen.Log("[WebSocket] error: " + ep.Msg)

	case "recv": // letter from other peer
		var rp RecvPacket
		err = json.Unmarshal(message, &rp)
		if err != nil {
			return
		}

		ws.peerID = rp.From
		ws.screen.Log(fmt.Sprintf("[WebSocket] got mail from %s: <%s> %d bytes", rp.From, rp.Msg.Topic, len(rp.Msg.Body)))

		switch rp.Msg.Topic {
		case "ping":
			ws.SendMessage(Message{"pong", ""})

		case "sdp":
			ws.rtc.SetRemoteSDP(rp.Msg.Body)

		case "candidate":
			ws.rtc.SetCandidate(rp.Msg.Body)
		}

	}
	return
}

func (ws *WebSocket) LoopMessage() {
	defer ws.conn.Close()
	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			ws.screen.Log("[WebSocket] read failed: " + err.Error())
			return
		}

		ws.screen.Log(fmt.Sprintf("[WebSocket] read: %d bytes", len(message)))

		err = ws.handleMessage(message)
		if err != nil {
			ws.screen.Log("[WebSocket] handle message failed: " + err.Error())
			return
		}
	}
}

func (ws *WebSocket) Connect(addr string) {
	ws.Reset()

	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}
	ws.screen.Log("[WebSocket] connecting to " + u.String())

	dialer := websocket.Dialer{
		ReadBufferSize:  65535,
		WriteBufferSize: 65535,
		NetDial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	}

	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		ws.screen.Log("[WebSocket] dial failed: " + err.Error())
		return
	}
	ws.conn = c

	// handle incoming signal
	go ws.LoopMessage()
	ws.rtc.Init()
}

// func (ws *WebSocket) Loop(quit chan struct{}) {
// 	interrupt := make(chan os.Signal, 1)
// 	signal.Notify(interrupt, os.Interrupt)

// 	defer ws.conn.Close()

// 	done := make(chan struct{})

// 	go ws.loopMessage(done)

// 	ticker := time.NewTicker(time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-quit:
// 			return

// 		case <-done:
// 			return

// 		case <-interrupt:
// 			ws.screen.Log("[WebSocket] got interrupt signal")

// 			err := ws.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
// 			if err != nil {
// 				ws.screen.Log("[WebSocket] write close failed:" + err.Error())
// 				return
// 			}
// 			select {
// 			case <-done:
// 			case <-time.After(time.Second):
// 			}
// 			return
// 		}
// 	}

// }
