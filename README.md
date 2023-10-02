# wschat
Websocket chat applications written in different languages and frameworks.

## The App
The app is a websocket server only which only does 3 things:
    1. Notify when a user connects to the chat.
    2. Notify when a user sends a chat.
    3. Notify when a user disconnects from the chat.
There is only 1, global, room. Users are assigned a UUID by the system.
The UUID is as follows: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
On join, users do not receive the chat history. Everything that goes on in the server can be seen by all, including all messages sent.

The programs themselves should take 1 command-line arg, the address.

## The Format
The message format sent to/from servers and clients is as follows:
```proto
type Message {
    // The sender of the message. "connect" and "disconnect" have this as "system".
    string sender
    // One of 4 possible strings: connect, chat, disconnect, error. "error" should only be sent by the server, after which the server should disconnect the client.
    string action
    // Contents of the message. This is the UUID of the user if the action is "connect" or "disconnect".
    string contents
    // A unix timestamp with second precision. Populated by the server on message receipt or right before a "system" message is sent. May not be populated on error messages.
    uint64 timestamp
}
```
As of this time, client may possibly forgo sending the sender?, action?, and timestamp fields.
A server should prefer sending a timestamp as a number, not a string. The client has been set up to handle parsing the timestamp into a bigint from a string or number.

## The Servers
The first message sent to a connecting client is a "connect" message, specifically, the same one that is echoed to the rest of the connected clients. After this first message is sent, the client can start receiving other messages.
If a bad message format is received on the websocket, the server should immediately close the connection (TODO: Send error message?).

# The Web Interface
Users can join via the web to any of the different servers, which will act as they're own chat rooms.

# TODO
- Send users chat history on join?
- Optimize Rust
