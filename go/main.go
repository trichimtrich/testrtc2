package main

import (
	"flag"
	"strings"

	"testrtc2/network"
	"testrtc2/screen"
)

func main() {
	addr := flag.String("addr", "192.168.1.104:6789", "websocket signal server")
	flag.Parse()

	quit := make(chan struct{})

	// terminal
	screen := screen.NewScreen()
	screen.Init()
	screen.SetTitle("Enter command ...")
	screen.Log("[System] Welcome to WebRTC Demo - Pion2 Golang")
	screen.Show()
	go screen.RenderLoop(quit)
	go screen.EventLoop(quit)

	// create websocket + webrtc
	ws := network.NewWebSocket(screen)
	rtc := network.NewWebRTC(screen)
	ws.SetWebRTC(rtc)
	rtc.SetWebSocket(ws)

	// register createOffer callback
	screen.RegisterCallback(func(data string) {
		if data == "/new" {
			// run in routine, dial might take long time
			// also init webrtc when establishing ws connection
			go ws.Connect(*addr)
		} else if strings.HasPrefix(data, "/peer ") {
			peerID := data[6:]
			ws.SetPeer(peerID)
		} else if data == "/media" {
			rtc.AddMedia()
		} else if data == "/data" {
			rtc.CreateDataChannel()
		} else if data == "/offer" {
			rtc.CreateOffer()
		} else {
			screen.Log("[System] Unknown command")
		}
		screen.SetBuffer("")
	})

	<-quit
	screen.Fini()
}
