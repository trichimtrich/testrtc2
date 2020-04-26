# testrtc2

Another test clients for [WebRTC](https://webrtc.org/), included multiple clients implemented in
- Vanila JS - works on all modern browsers (Chromium + Firefox + Safari)
- Golang with [pion](https://github.com/pion/webrtc) library - works on any Desktop OS
- Flutter/Dart lang with [Flutter-WebRTC](https://pub.dev/packages/flutter_webrtc) wrapper - works on Mobile (iOS + Android) + Chromium

The old repo [testrtc](https://github.com/trichimtrich/testrtc) was made to do blackbox connectivity test/experiment between nodes (behide NAT, dockerize environment, other locations, ...)

With this repo, we aim to experiment the compatibility/functionality between clients.

## Features

Can break down each clients to parts:
- Connectivity:
    - Signaling (via websocket)
    - Init / close peer connection
- Transceiver (transport unit): [Link](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/addTransceiver)
    - Add on kinds: audio / video
    - Add on directions: sendrecv / sendonly / recvonly
- SDP role:
    - CreateOffer -> local SDP: [Link](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/createOffer)
    - Options for CreateOffer
    - CreateAnswer -> local SDP: [Link](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/createAnswer)
    - Send local SDP to peer (-> set remote SDP)
- ICE gathering:
    - Capture ICE Candidate from STUN / TURN: [Link](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/onicecandidate)
    - Send local ICE Candidate to peer (via signal)
    - Receive remote ICE Candidate from peer (via signal)
    - Add remote ICE Candidate from peer: [Link](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/addIceCandidate)
    - Restart ICE: [Link](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/restartIce)
- Media stream:
    - Add sample video track: [Link1](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/addTrack) [Link2](https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement/captureStream)
    - Add sample audio track: [Link1](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/addTrack) [Link2](https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement/captureStream)
    - Add webcam track: [Link1](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/addTrack) [Link2](https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getUserMedia)
    - Add microphone track: [Link1](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/addTrack) [Link2](https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getUserMedia)
    - Remove all added tracks: [Link](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/removeTrack)
- Data channel:
    - Add data channel: [Link](https://developer.mozilla.org/en-US/docs/Web/API/RTCPeerConnection/createDataChannel)
    - Close all data channels

With these breakdown parts we can do blackbox test to watch behavior of each clients:
- What if we set local and remote SDP but don't add ICE
- What if we set remote SDP twice
- What if we send tracks to peer without transceiver
- What if we add media track before createOffer
- What if we add media track after createAnswer
- What if we close data channel
- ...

## Components

```
.
├── README.md
├── flutter     --> flutter client
├── go          --> golang client
├── signal.py   --> signal server
├── web.html    --> web client
```

### Signal server

Very simple websocket implementation in Python
- Assign an ID for each connection (use as userID, peerID in client)
- Forward message between IDs

Install
```bash
$ pip install websocket_server
```

Run
```bash
$ python signal.py HOST PORT
$ python signal.py 127.0.0.1 6789
```

### Web client

Full features, can run `web.html` directly in browser on multiple platforms. Or you can serve it via a webserver (some browsers might need a web origin)

Example:
```bash
$ python -m http.server PORT --bind HOST
```

### Golang client

Golang version works in interactive terminal graphic, and only implemented some parts
- connectivity
- create and send offer
- automatic answer from remote SDP
- media tracks (gststreamer)
- data channel

Needs to install `gststreamer` dependency first for this to work.

```bash
# Debian / Ubuntu
$ sudo apt-get install libgstreamer1.0-dev libgstreamer-plugins-base1.0-dev gstreamer1.0-plugins-good

# Windows MinGW64/MSYS2
$ pacman -S mingw-w64-x86_64-gstreamer mingw-w64-x86_64-gst-libav mingw-w64-x86_64-gst-plugins-good mingw-w64-x86_64-gst-plugins-bad mingw-w64-x86_64-gst-plugins-ugly

# macOS
$ brew install gst-plugins-good gst-plugins-ugly pkg-config && export PKG_CONFIG_PATH="/usr/local/opt/libffi/lib/pkgconfig"
```

Run
```bash
$ cd go
$ go run . -addr SIGNALSV

# example
$ go run . -addr 127.0.0.1:6789
```

If this somehow `panic`, please use `reset` command to reset terminal graphic

### Flutter/dart client

Flutter/dart version only acts as receiver, no added media track support

- Install flutter SDK first depends on your platform : [Link](https://flutter.dev/docs/get-started/install)

- Run flutter
    - iOS/Android:
    ```bash
    $ cd flutter
    $ flutter run
    ```
    - Chromium: 
    ```bash
    $ cd flutter
    $ flutter run -d chrome
    ```

Flutter is supposed to be `write once, compile and run anywhere`. You can compile this source code to standalone app easily with flutter -> [Link](https://flutter.dev/docs/testing/build-modes)

This client uses a wrapper for WebRTC
- iOS/Android: binding for libWebRTC -> compile to native app
- Web: binding for WebRTC JS API -> compile to web app

If you are facing the problem run in Firefox or Safari with this flutter web app, because there is an issue with `dart:html` (by the commit timestamp of this README)
- https://github.com/dart-lang/sdk/issues/38787

## Notes

This repo is used for blackbox testing between clients, not to claim and disclaim anything. Here is my personal results:
- First call of CreateOffer and CreateAnswer will immediately create lots of local ICE candidates (from STUN/TURN).
- Remote ICE candidates should be added only after remote SDP is set.
- ICE candidates should be transfer at least from 1 peer to another to make successful connection. But very stable when both parties exchange their local ICE candidates.
- Add transreceiver is not neccessary needed to send/recv tracks.
- After successful connection between parties, any party can createOffer to other.
- Media provider should be the one createOffer.
- Media track should be added before createOffer.
- Added media tracks will not send immediately, need renegotiation (createOffer -> createAnswer).
- Same thing with DataChannel
- Restart Ice does not work

## TODO

- setConfiguration for peerConnection
- DataChannel with name
- send message via DataChannel
- replaceTrack
- codec(?)
- view and edit SDP(?)