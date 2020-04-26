from websocket_server import WebsocketServer
import json
import sys

users = {}
idx_map = {}
count = 0

# Called for every client connecting (after handshake)
def new_client(client, server):
    global count, users, idx_map
    idx = str(count + 1)
    count += 1
    print('New user: {}'.format(idx))
    users[idx] = client
    server.send_message(client, json.dumps({
        'action': 'init',
        'id': idx,
    }))

def get_client_idx(client):
    global users
    for k, cl in users.items():
        if cl == client:
            return k

# Called for every client disconnecting
def client_left(client, server):
    global users
    k = get_client_idx(client)
    del users[k]
    print('Client {} disconnected'.format(k))

# Called when a client sends a message
def message_received(client, server, message):
    data = json.loads(message)
    if data['action'] == 'send':
        from_id = get_client_idx(client)
        to_id = data['to']
        if to_id not in users:
            print("Client {} sent bad request".format(from_id))
            server.send_message(
                client,
                json.dumps({
                    'action': 'error',
                    'msg': 'Cannot find peer {}'.format(repr(to_id))
                })
            )
        else:
            server.send_message(
                users[to_id],
                json.dumps({
                    'action': 'recv',
                    'from': from_id,
                    'msg': data['msg'],
                })
            )
    else:
        print("unsupported event: {}", data)

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print('Usage: python {} HOST PORT'.format(sys.argv[0]))
        sys.exit(1)

    HOST = sys.argv[1]
    PORT = int(sys.argv[2])
    server = WebsocketServer(PORT, host=HOST)
    server.set_fn_new_client(new_client)
    server.set_fn_client_left(client_left)
    server.set_fn_message_received(message_received)
    print('Server is running at {}:{}'.format(HOST, PORT))
    server.run_forever()