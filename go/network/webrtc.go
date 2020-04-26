package network

import (
	"fmt"
	"math/rand"

	"testrtc2/gst"
	"testrtc2/screen"

	"github.com/pion/webrtc/v2"
)

type WebRTC struct {
	screen     *screen.Screen
	ws         *WebSocket
	conn       *webrtc.PeerConnection
	isOffering bool
	isPeered   bool
	pipes      []*gst.Pipeline
}

func NewWebRTC(screen *screen.Screen) *WebRTC {
	return &WebRTC{screen, nil, nil, false, false, nil}
}

func (rtc *WebRTC) SetWebSocket(ws *WebSocket) {
	rtc.ws = ws
}

func (rtc *WebRTC) StopPipe() {
	// stop pipe
	for _, pipe := range rtc.pipes {
		pipe.Stop()
		rtc.screen.Log("[Track] Stop a pipe")
	}
	rtc.pipes = nil
}

func (rtc *WebRTC) Reset() {
	rtc.StopPipe()
	rtc.isOffering = false
	rtc.isPeered = false

	if rtc.conn != nil {
		rtc.conn.Close()
		rtc.screen.Log("[System] Close previous peer connection")
		rtc.conn = nil
	}
}

func (rtc *WebRTC) registerDataCallback(channel *webrtc.DataChannel) {
	channel.OnOpen(func() {
		rtc.screen.Log("[DataChannel] OnOpen")
		channel.SendText("ping")
	})

	channel.OnError(func(err error) {
		rtc.screen.Log("[DataChannel] OnError: " + err.Error())
	})

	channel.OnClose(func() {
		rtc.screen.Log("[DataChannel] OnClose")
	})

	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		if msg.IsString {
			st := string(msg.Data)
			rtc.screen.Log("[DataChannel] OnMessage: " + st)
			if st == "ping" {
				channel.SendText("pong")
			}
		} else {
			// pass
		}
	})

	rtc.screen.Log("[DataChannel] Register callbacks")
}

func (rtc *WebRTC) CreateDataChannel() {
	if rtc.conn == nil {
		rtc.screen.Log("[System] No peer connection to create data channel")
		return
	}

	// DataChannel
	channel, err := rtc.conn.CreateDataChannel("data", nil)
	if err != nil {
		rtc.screen.Log("[WebRTC] create data channel failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] Create Data Channel")

	rtc.registerDataCallback(channel)
}

func (rtc *WebRTC) AddMedia() {
	if rtc.conn == nil {
		rtc.screen.Log("[System] No peer connection to add media")
		return
	}

	// stop pipelines
	rtc.StopPipe()

	// remove all media sender
	for _, sender := range rtc.conn.GetSenders() {
		rtc.screen.Log("[WebRTC] Remove track - " + sender.Track().Label())
		rtc.conn.RemoveTrack(sender)
	}

	// Audio Track
	opusTrack, err := rtc.conn.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "pion1")
	if err != nil {
		rtc.screen.Log("[WebRTC] create new audio track failed: " + err.Error())
		return
	}

	rtc.screen.Log("[WebRTC] create new audio track")

	_, err = rtc.conn.AddTrack(opusTrack)
	if err != nil {
		rtc.screen.Log("[WebRTC] add new audio track failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] add new audio track")

	// Video Track
	vp8Track, err := rtc.conn.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "pion2")
	if err != nil {
		rtc.screen.Log("[WebRTC] create new video track failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] create new video track")

	_, err = rtc.conn.AddTrack(vp8Track)
	if err != nil {
		rtc.screen.Log("[WebRTC] add new video track failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] add new video track")

	// build pipelines
	audioSrc := "audiotestsrc ! audioconvert ! queue"
	videoSrc := "videotestsrc pattern=snow ! video/x-raw,width=320,height=240 ! queue"

	pipe1 := gst.CreatePipeline(webrtc.Opus, []*webrtc.Track{opusTrack}, audioSrc)
	pipe2 := gst.CreatePipeline(webrtc.VP8, []*webrtc.Track{vp8Track}, videoSrc)
	rtc.pipes = append(rtc.pipes, pipe1, pipe2) // lazy, have to remove pipes first

	// starting...
	pipe1.Start()
	pipe2.Start()

	// if rtc.isPeered {
	// 	// already in connection, re-offer?
	// 	rtc.CreateOffer()
	// }
}

func (rtc *WebRTC) CreateOffer() {
	if rtc.conn == nil {
		rtc.screen.Log("[System] No peer connection to make offer")
		return
	}

	if rtc.ws.GetPeer() == "" {
		rtc.screen.Log("[System] Need to set peer ID first")
		return
	}

	rtc.isOffering = true

	// Create Offer
	offer, err := rtc.conn.CreateOffer(nil)
	if err != nil {
		rtc.screen.Log("[WebRTC] create offer failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] create offer")

	rtc.setLocalSDP(offer)
}

func (rtc *WebRTC) setLocalSDP(desc webrtc.SessionDescription) {
	err := rtc.conn.SetLocalDescription(desc)
	if err != nil {
		rtc.screen.Log("[WebRTC] set local sdp failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] local sdp set")

	sdp, err := Encode(desc)
	if err != nil {
		rtc.screen.Log("[WebRTC] sdp encode failed: " + err.Error())
		return
	}

	rtc.ws.SendMessage(Message{"sdp", sdp})
}

func (rtc *WebRTC) createAnswer() {
	answer, err := rtc.conn.CreateAnswer(nil)
	if err != nil {
		rtc.screen.Log("[WebRTC] create answer failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] answer created")

	rtc.setLocalSDP(answer)
}

func (rtc *WebRTC) SetRemoteSDP(sdp string) {
	desc := webrtc.SessionDescription{}
	err := Decode(sdp, &desc)
	if err != nil {
		rtc.screen.Log("[WebRTC] sdp decode failed: " + err.Error())
		return
	}

	err = rtc.conn.SetRemoteDescription(desc)
	if err != nil {
		rtc.screen.Log("[WebRTC] set remote sdp failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] remote sdp set")

	if rtc.isOffering {
		rtc.isOffering = false
	} else {
		// suppose to answer if not in offering mode
		rtc.createAnswer()
	}
}

func (rtc *WebRTC) SetCandidate(data string) {
	var ice webrtc.ICECandidateInit
	err := Decode(data, &ice)
	if err != nil {
		rtc.screen.Log("[WebRTC] decode IceCandidate failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] decoded IceCandidate: " + ice.Candidate)

	err = rtc.conn.AddICECandidate(ice)
	if err != nil {
		rtc.screen.Log("[WebRTC] add IceCandidate failed: " + err.Error())
		return
	}
	rtc.screen.Log("[WebRTC] add IceCandidate from peer")
}

func (rtc *WebRTC) Init() {
	rtc.Reset()

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	rr, err := webrtc.NewPeerConnection(config)
	if err != nil {
		rtc.screen.Log("[WebRTC] create peer connection failed: " + err.Error())
		return
	}
	rtc.conn = rr

	// // Allow us to receive 1 audio track, and 1 video track
	// if _, err = rtc.conn.AddTransceiver(webrtc.RTPCodecTypeAudio); err != nil {
	// 	rtc.screen.Log("[WebRTC] AddTransceiver Audio failed")
	// }
	// if _, err = rtc.conn.AddTransceiver(webrtc.RTPCodecTypeVideo); err != nil {
	// 	rtc.screen.Log("[WebRTC] AddTransceiver Video failed")
	// }

	rtc.screen.Log("[WebRTC] peer connection created")

	rtc.conn.OnSignalingStateChange(func(state webrtc.SignalingState) {
		rtc.screen.Log("[WebRTC] OnSignalingStateChange -> " + state.String())
	})

	rtc.conn.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		rtc.screen.Log("[WebRTC] OnICEConnectionStateChange -> " + state.String())
	})

	rtc.conn.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		rtc.screen.Log("[WebRTC] OnICEGatheringStateChange -> " + state.String())
	})

	rtc.conn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		rtc.screen.Log("[WebRTC] OnConnectionStateChange -> " + state.String())
		if state == webrtc.PeerConnectionStateConnected {
			// peer established
			rtc.isPeered = true
		}
	})

	rtc.conn.OnTrack(func(track *webrtc.Track, rec *webrtc.RTPReceiver) {
		rtc.screen.Log(fmt.Sprintf("[WebRTC] OnTrack -> %s - %s", track.Kind().String(), track.ID()))
	})

	rtc.conn.OnDataChannel(func(channel *webrtc.DataChannel) {
		rtc.screen.Log("[WebRTC] OnDataChannel: " + channel.Label())
		rtc.registerDataCallback(channel)
	})

	rtc.conn.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		// TODO: send via websocket
		if candidate != nil {
			rtc.screen.Log("[WebRTC] OnICECandidate: " + candidate.String())
			body, err := Encode(candidate.ToJSON())
			if err != nil {
				rtc.screen.Log("[WebRTC] encode IceCandidate failed")
				return
			}
			rtc.ws.SendMessage(Message{"candidate", string(body)})
		}
	})
}
