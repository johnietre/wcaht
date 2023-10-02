// Parses the entire message JSON, used to parse the timestamp into a bigint
function parseMessage(str) {
  const re = /,?\s*"timestamp"\s*:"?\W*(\d+)"?\s*/;
  const matches = str.match(re);
  const ret = {};
  if (matches !== null) {
    ret.timestamp = BigInt(matches[1]);
    str = str.replace(matches[0], "");
  }
  return Object.assign(ret, JSON.parse(str));
}

const App = {
  data() {
    return {
      ws: null,
      isOpen: false,

      uuid: "",
      servers: [],
      messages: [],
      newMessages: false,

      server: "",
      messageContents: "",
      scrollToBottom: true,
    };
  },

  mounted() {
    this.refreshServers();
  },
  updated() {
    this.goToBottom();
  },

  methods: {
    connectToWs(addr) {
      if (this.isConnected()) {
        this.reset();
        alert("Disconnected from server");
      }
      try {
        this.ws = new WebSocket(addr);
        this.ws.onopen = this.openHandler;
        this.ws.onmessage = this.messageHandler;
        this.ws.onerror = this.errorHandler;
        this.ws.onclose = this.closeHandler;
      } catch (e) {
        console.log(`error connecting to server: ${e}`);
        alert("An error occurred: could not connect to server");
      }
    },
    sendMsg(msgStr) {
      if (!this.isConnected()) {
        alert("Cannot send message: not connected to a server");
        return;
      }
      const msg = {sender: this.uuid, action: "chat", contents: msgStr};
      this.ws.send(JSON.stringify(msg))
      this.messageContents = "";
    },
    openHandler() {
      this.isOpen = true;
    },
    messageHandler(wsMsg) {
      let msg;
      try {
        //msg = JSON.parse(wsMsg.data);
        msg = parseMessage(wsMsg.data);
      } catch (e) {
        console.log(wsMsg);
        this.errorHandler(`error parsing message: ${e}`);
        return;
      }
      switch (msg.action) {
        case "connect":
          if (this.uuid === "") {
            this.uuid = msg.contents;
          }
          break;
        case "chat":
          break;
        case "disconnect":
          if (msg.contents === this.uuid) {
            this.reset();
            alert("Disconnected from server")
          }
          break;
        case "error":
          this.errorHandler(`error from server: ${msg.contents}`);
          break;
        default:
          this.errorHandler(`bad message received from server: ${wsMsg.data}`);
          return;
      }
      const index = this.messages.findLastIndex((elem) => elem.timestamp < msg.timestamp);
      this.messages.splice(index + 1, 0, msg);
      this.newMessages = true;
    },
    goToBottom() {
      if (!this.newMessages) {
        return;
      }
      this.newMessages = false;
      if (!this.scrollToBottom) {
        return;
      }
      const msgsDiv = document.querySelector("#messages-div");
      if (msgsDiv === null) {
        return;
      }
      msgsDiv.scrollTop = msgsDiv.scrollHeight;
    },
    errorHandler(errMsg) {
      this.reset();
      console.log(`an error occurred:`, errMsg);
      alert("An error occurred: disconnecting...");
    },
    closeHandler() {
      // Websocket hasn't been removed from app
      if (this.isConnected()) {
        this.reset();
        alert("Disconnected from server");
      }
    },
    changeServer() {
      //const addr = this.servers[event.target.value];
      const addr = this.servers[this.server];
      if (addr === undefined) {
        this.reset();
        return;
      }
      this.connectToWs(addr);
    },
    reset() {
      if (this.isConnected()) {
        this.ws.close();
        this.isOpen = false;
      }
      this.uuid = "";
      this.messages = [];
      this.ws = null;
    },
    isConnected() {
      return this.isOpen && this.ws !== null;
    },
    async refreshServers() {
      try {
        const url = new URL("/servers", document.location.href);
        const resp = await fetch(url);
        if (!resp.ok) {
          console.log(`an error response was returned after refreshing servers:`, resp);
          alert("An error occurred...");
          return;
        }
        this.servers = await resp.json();
      } catch (e) {
        console.log(`an error occurred while attempting to refresh servers: ${e}`);
        alert("An error occurred...");
        return;
      }
    }
  }
};

Vue.createApp(App).mount("#app");
