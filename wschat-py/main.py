#!/usr/bin/env python3
import message, sys, uuid as uuidpkg, uvicorn
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from json.decoder import JSONDecodeError

app = FastAPI()

class Clients:
    def __init__(self):
        self._clients = []
        """
        self._clients = set([])
        self._updated = False
        self._iter_clients = set([])
        """

    def add(self, ws: WebSocket):
        self._clients.append(ws)
        """
        self._updated = True
        self._clients.add(ws)
        """

    def remove(self, ws: WebSocket):
        self._clients.remove(ws)
        """
        self._updated = True
        self._clients.remove(ws)
        """

    #def iter(self) -> set[WebSocket]:
    def iter(self) -> list[WebSocket]:
        return self._clients
        """
        if self._updated:
            self._updated = False
            self._iter_clients = self._clients.copy()
        return self._iter_clients
        """

clients = Clients()
#clients: set[WebSocket] = set()
# dict[UUID, WebSocket]
#clients: dict[str, WebSocket] = dict()

@app.get("/num")
async def num_clients():
    return len(clients.iter())

@app.websocket("/")
async def handler(ws: WebSocket):
    await ws.accept()
    uuid = str(uuidpkg.uuid4())

    msg = message.Message.new_system(message.ACTION_CONNECT, uuid)
    msg_json = msg.to_json()
    # Don't add ws to clients until after sending connect os that messages
    # aren't received before the connect is sent to all.
    await broadcast_msg_str(msg_json)
    await ws.send_text(msg_json)
    clients.add(ws)

    while True:
        try:
            msg_json = await ws.receive_text()
            msg = message.Message.from_json(msg_json)
            await broadcast_msg(message.Message.new_chat(uuid, msg.contents))
        except JSONDecodeError:
            await ws.send_text(
                message.Message.new_system(
                    message.ACTION_ERROR,
                    "bad message",
                ),
            )
        except WebSocketDisconnect:
            break
        except Exception as e:
            #print(f"[|{uuid}] error reading from client: {e}")
            break

    msg = message.Message.new_system(message.ACTION_DISCONNECT, uuid)
    await broadcast_msg(msg)
    clients.remove(ws)


async def broadcast_msg(msg: message.Message):
    await broadcast_msg_str(msg.to_json())

async def broadcast_msg_str(s: str):
    for ws in clients.iter():
        try: await ws.send_text(s)
        except: pass

def fatal(msg: str):
    print(msg)
    exit(1)

if __name__ == "__main__":
    # TODO: Workers
    if len(sys.argv) != 2:
        fatal("must provide the address (and only the address)")
    addr = sys.argv[1]
    host, partition, port = addr.rpartition(":")
    if partition == "":
        fatal("invalid address provided")
    try: port = int(port)
    except: fatal("invalid port")
    print(f"Listening on {addr}")
    uvicorn.run(app, host=host, port=port, log_level="warning")
