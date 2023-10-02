import express from "express";
import * as http from "http";
import { v4 as uuidpkg } from "uuid";
import * as WebSocket from "ws";

const app = express();
const map = new Map();

const srvr = http.createServer(app);

const wsSrvr = new WebSocketServer({ server });

wsSrvr.on("connection", (ws: WebSocket) {
  const uuid = uuidpkg.v4();
});

const handler = (ws: WebSocket) => {
  const uuid = uuidpkg.v4();
};
