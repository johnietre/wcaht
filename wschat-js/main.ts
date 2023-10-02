import express from "express";
import * as http from "http";
import { v4 as uuidpkg } from "uuid";
import * as WebSocket from "ws";

const app = express();
const clients = new Map();

const srvr = http.createServer(app);

const wsSrvr = new WebSocketServer({ server });

wsSrvr.on("connection", (ws: WebSocket) {
  const uuid = uuidpkg.v4();
});

const handler = (ws: WebSocket) => {
  const uuid = uuidpkg.v4();
};

const broadcastMsg = (msg: Message) => {
  //
};

type Action = "connect" | "chat" | "disconnect" | "error";

type Message = {
  sender: string;
  action: Action;
  contents: string;
  timestamp: bigint;
};

function NewChatMessage(sender: string, contents: string): Message {
}

function parseMessage(str: string): Message {
}

function stringifyMessage(msg: Message): string {
}
