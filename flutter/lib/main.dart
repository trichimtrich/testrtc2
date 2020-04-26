import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_webrtc/webrtc.dart';
import 'websocket.dart' if (dart.library.js) 'websocket_web.dart';

void main() {
  runApp(MyApp());
}

class MyApp extends StatelessWidget {
  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Flutter Demo',
      theme: ThemeData(
        primarySwatch: Colors.blue,
        visualDensity: VisualDensity.adaptivePlatformDensity,
      ),
      home: MyHomePage(),
    );
  }
}

class MyHomePage extends StatefulWidget {
  MyHomePage();

  @override
  _MyHomePageState createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  String server = 'ws://192.168.1.102:6789/';

  SimpleWebSocket socket;

  RTCPeerConnection pc;

  MediaStream ms;
  RTCVideoRenderer screen = new RTCVideoRenderer();
  bool isOfferer = false;

  String userID, peerID, stState, stAudio, stVideo, stData;

  BuildContext _context;

  Future<void> popup(String title, String msg) async {
    showDialog(
      context: _context,
      builder: (_) => AlertDialog(
        title: Text(title),
        content: Text(msg),
      )
    );
    log(msg);
  }

  void log(String msg) {
    print(msg);
    // TODO: memo widget?
  }

  Future<void> reset() async {
    if (ms != null) {
      screen.srcObject = null;
      await ms.dispose();
      ms = null;
      log('[System] Reset media objects');
    }

    if (pc != null) {
      await pc.close();
      log('[System] Close previous peer connection');
      pc = null;
    }

    if (socket != null) {
      socket.close();
      log('[System] Close previous web socket');
      socket = null;
    }

    userID = peerID = stState = stAudio = stVideo = stData = null;
    isOfferer = false;

    setState(() {});
  }

  void sendMessage(msg) {
    socket.send(jsonEncode({
      'action': 'send',
      'to': peerID,
      'msg': msg,
    }));
    log('[WebSocket] Send message <${msg['topic']}> to peer $peerID');
  }

  void handleCommand(packet) {
    Map data = jsonDecode(packet);

    switch (data['action']) {
      case 'init':
        userID = data['id'];
        log('[WebSocket] My ID - $userID');
        setState(() {});
        break;

      case 'recv':
        peerID = data['from'];
        setState(() { });
        var msg = data['msg'];

        log('[WebSocket] Get message from peer $peerID: <${msg['topic']}> ${msg['body'].length} bytes');
        switch (msg['topic']) {
          case 'ping':
            sendMessage({
              'topic': 'pong',
              'body': '',
            });
            break;

          case 'sdp':
            setRemoteSDP(msg['body']);
            break;

          case 'candidate':
            Map c = jsonDecode(utf8.decode(base64Decode(msg['body'])));
            // TODO: should buffer + error catch
            pc.addCandidate(RTCIceCandidate(
              c['candidate'],
              c.containsKey('sdpMid') ? c['sdpMid'] : '',
              c.containsKey('sdpMLineIndex') ? c['sdpMLineIndex'] : 0,
            ));
            log('[WebRTC] add IceCandidate from peer');
            break;
        }

        break;

      case 'error':
        log('[WebSocket] Error: ${data['msg']}');
        break;
    }
  }

  Future<void> initNetwork() async {
    await reset();

    // WebSocket
    String _server = server.replaceAll("ws:", "http:");
    socket = SimpleWebSocket(_server);

    socket.onOpen = () {
      log('[WebSocket] onOpen');
    };

    socket.onClose = (int code, String reason) {
      log('[WebSocket] onClose [$code => $reason]');
    };

    socket.onMessage = (message) {
      handleCommand(message);
    };

    await socket.connect();
    log('[WebSocket] Connected');

    // WebRTC
    pc = await createPeerConnection({
      'iceServers': [
        {'urls': 'stun:stun.l.google.com:19302'},
      ],
      'sdpSemantics': 'unified-plan' // for pion
    }, {
      "mandatory": {},
      "optional": [
        {"DtlsSrtpKeyAgreement": true},
      ],
    });
    log('[WebRTC] Create Peer Connection');

    pc.onIceCandidate = (RTCIceCandidate candidate) {
      log('[WebRTC] onIceCandidate: ${candidate.candidate}');

      sendMessage({
        'topic': 'candidate',
        'body': base64Encode(utf8.encode(jsonEncode(candidate.toMap())))
      });
    };

    pc.onIceConnectionState = (RTCIceConnectionState state) {
      switch (state) {
        case RTCIceConnectionState.RTCIceConnectionStateChecking:
          stState = 'checking';
          break;
        case RTCIceConnectionState.RTCIceConnectionStateClosed:
          stState = 'closed';
          break;
        case RTCIceConnectionState.RTCIceConnectionStateCompleted:
          stState = 'completed';
          break;
        case RTCIceConnectionState.RTCIceConnectionStateConnected:
          stState = 'connected';
          break;
        case RTCIceConnectionState.RTCIceConnectionStateDisconnected:
          stState = 'disconnected';
          break;
        case RTCIceConnectionState.RTCIceConnectionStateFailed:
          stState = 'failed';
          break;
        default:
          stState = 'n/a';
      }
      log('[WebRTC] onIceConnectionState - $stState');
      setState(() {});
    };

    pc.onIceGatheringState = (state) {
      log('[WebRTC] onIceGatheringState - $state');
    };

    pc.onAddTrack = (MediaStream stream, MediaStreamTrack track) {
      log('[WebRTC] onAddTrack: ${stream.id} - ${track.id} - ${track.kind}');
      if (track.kind == 'audio') {
        stAudio = 'true';
      }
      if (track.kind == 'video') {
        stVideo = 'true';
      }
      setState(() {});
    };

    pc.onAddStream = (MediaStream stream) {
      log('[WebRTC] onAddStream: ${stream.id}');
      if (ms == null) {
        ms = stream;
      } else {
        stream.getAudioTracks().forEach((track) => ms.addTrack(track, addToNative: false ));
        stream.getVideoTracks().forEach((track) => ms.addTrack(track, addToNative: false ));
      }
      
      if (stream.getAudioTracks().length > 0) {
        stAudio = 'true';
      }
      if (stream.getVideoTracks().length > 0) {
        stVideo = 'true';
      }
      setState(() {
        screen.srcObject = ms;
      });
    };

    pc.onDataChannel = (RTCDataChannel channel) {
      log('[WebRTC] onDataChannel ');
      registerDataCallback(channel);
      // recv channel => auto true
      stData = 'true';
      setState(() {});
    };

    popup('info', 'created new peer connection');
  }

  void registerDataCallback(RTCDataChannel channel) {
    channel.onMessage = (RTCDataChannelMessage msg) {
      log('[DataChannel] onMessage: ${msg.text}');

      if (msg.text == 'ping') {
        channel.send(RTCDataChannelMessage('pong'));
      }
    };

    channel.onDataChannelState = (RTCDataChannelState state) {
      log('[DataChannel] onDataChannelState: $state');
      if (state == RTCDataChannelState.RTCDataChannelOpen) {
        // its open => true
        channel.send(RTCDataChannelMessage('ping'));
        stData = 'true';
        setState(() {});
      }
    };

    log('[DataChannel] Register callbacks');
  }

  Future<void> addMedia() async {
    if (pc == null) {
      popup('Error', 'no peer connection');
      return;
    }

    popup('info', 'We dont implement yet');
    log('[WebRTC] Add media');

    return;
  }

  Future<void> createDataChannel() async {
    if (pc == null) {
      popup('Error', 'no peer connection');
      return;
    }

    RTCDataChannel channel;
    try {
      channel = await pc.createDataChannel('my-data-channel', RTCDataChannelInit());
    } catch (e) {
      popup('error', '[WebRTC] Create data channel failed: $e');
      return;
    }

    popup('info', '[WebRTC] Created data channel');
    registerDataCallback(channel);
  }

  Future<void> setRemoteSDP(String remoteSDP) async {
    Map d = jsonDecode(utf8.decode(base64Decode(remoteSDP)));
    RTCSessionDescription sdp = RTCSessionDescription(d['sdp'], d['type']);

    try {
      await pc.setRemoteDescription(sdp);
    } catch (e) {
      popup('error', '[WebRTC] Set Remote Description failed: $e');
      return;
    }

    log('[WebRTC] Set Remote Description');

    if (isOfferer) {
      isOfferer = false;
    } else {
      await createAnswer();
    }
  }

  Future<void> setLocalSDP(RTCSessionDescription sdp) async {
    try {
      await pc.setLocalDescription(sdp);
    } catch (e) {
      popup('error', '[WebRTC] Set Local Description failed: $e');
      return;
    }

    log('[WebRTC] Set Local Description');

    sendMessage({
      'topic': 'sdp',
      'body': base64Encode(utf8.encode(jsonEncode(sdp.toMap()))),
    });
  }

  Future<void> createAnswer() async {
    log('[WebRTC] Creating answer');

    RTCSessionDescription answer = await pc.createAnswer({
      'mandatory': {
        'OfferToReceiveAudio': true,
        'OfferToReceiveVideo': true,
      },
      'optional': [],
    });
    log('[WebRTC] Create Answer');

    popup('info', 'created answer');

    await setLocalSDP(answer);
  }

  Future<void> createOffer() async {
    if (pc == null) {
      popup('Error', 'no peer connection');
      return;
    }

    isOfferer = true;

    log('[WebRTC] Creating offer');

    RTCSessionDescription offer = await pc.createOffer({
      'offerToReceiveAudio': true,
      'offerToReceiveVideo': true,
    });
    log('[WebRTC] Create Offer');
    popup('info', 'created offer');

    await setLocalSDP(offer);
  }

  initScreen() async {
    await screen.initialize();
  }

  @override
  void initState() {
    super.initState();
    initScreen();
  }

  @override
  void deactivate() {
    super.deactivate();
    screen.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _context = context;
    return Material(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: <Widget>[
          Text('My ID: $userID - Peer ID: $peerID'),
          Text('WebRTC state: $stState'),
          Text('Audio: $stAudio - Video: $stVideo'),
          Text('Data Channel: $stData'),
          Container(
            width: 320,
            height: 240,
            child: RTCVideoView(screen),
            decoration: BoxDecoration(
              color: Color.fromRGBO(255, 255, 0, 1),
            ),
          ),
          Container(
            width: 320,
            child: Column(
              children: <Widget>[
                Text(''),
                TextField(
                  decoration: InputDecoration(
                    border: OutlineInputBorder(),
                    labelText: 'Signal server',
                  ),
                  controller: TextEditingController(text: server),
                  onChanged: (String str) {
                    server = str;
                  },
                ),
                ButtonBar(
                  alignment: MainAxisAlignment.center,
                  children: <Widget>[
                    FlatButton(
                      color: Colors.blue,
                      textColor: Colors.white,
                      disabledColor: Colors.grey,
                      disabledTextColor: Colors.black,
                      padding: EdgeInsets.all(8.0),
                      splashColor: Colors.blueGrey,
                      onPressed: () {
                        initNetwork();
                      },
                      child: Text(
                        "New RTC",
                        style: TextStyle(fontSize: 20.0),
                      ),
                    ),
                    FlatButton(
                      color: Colors.blue,
                      textColor: Colors.white,
                      disabledColor: Colors.grey,
                      disabledTextColor: Colors.black,
                      padding: EdgeInsets.all(8.0),
                      splashColor: Colors.blueGrey,
                      onPressed: () {
                        reset();
                      },
                      child: Text(
                        "Disconnect",
                        style: TextStyle(fontSize: 20.0),
                      ),
                    )
                  ],
                ),
              ],
            ),
          ),
          Container(
            width: 320,
            child: Column(
              children: <Widget>[
                TextField(
                  decoration: InputDecoration(
                    border: OutlineInputBorder(),
                    labelText: 'Peer ID',
                  ),
                  controller: TextEditingController(text: peerID),
                  onChanged: (String str) {
                    peerID = str;
                  },
                ),
                FlatButton(
                  color: Colors.blue,
                  textColor: Colors.white,
                  disabledColor: Colors.grey,
                  disabledTextColor: Colors.black,
                  padding: EdgeInsets.all(8.0),
                  splashColor: Colors.blueGrey,
                  onPressed: () {
                    createOffer();
                  },
                  child: Text(
                    "Create Offer",
                    style: TextStyle(fontSize: 20.0),
                  ),
                )
              ],
            ),
          ),
          ButtonBar(
            alignment: MainAxisAlignment.center,
            children: <Widget>[
              FlatButton(
                color: Colors.blue,
                textColor: Colors.white,
                disabledColor: Colors.grey,
                disabledTextColor: Colors.black,
                padding: EdgeInsets.all(8.0),
                splashColor: Colors.blueGrey,
                onPressed: () {
                  addMedia();
                },
                child: Text(
                  "Add Media",
                  style: TextStyle(fontSize: 20.0),
                ),
              ),
              FlatButton(
                color: Colors.blue,
                textColor: Colors.white,
                disabledColor: Colors.grey,
                disabledTextColor: Colors.black,
                padding: EdgeInsets.all(8.0),
                splashColor: Colors.blueGrey,
                onPressed: () {
                  createDataChannel();
                },
                child: Text(
                  "Add DataChannel",
                  style: TextStyle(fontSize: 20.0),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
