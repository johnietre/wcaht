package main

import (
  "encoding/json"
  "errors"
  "fmt"
  "io"
  "log"
  "net/http"
  _ "net/http/pprof"
  "os"
  "sync"
  "sync/atomic"
  //"time"

  uuidpkg "github.com/google/uuid"
  webs "golang.org/x/net/websocket"
  "wschat/wschat-go/common"
)

type Channel[T any] struct {
  c chan T
  closed atomic.Bool
  mtx sync.RWMutex
}

func NewChannel[T any](l int) *Channel[T] {
  return &Channel[T]{c: make(chan T, l)}
}

func (c *Channel[T]) Send(val T) {
  c.mtx.RLock()
  if !c.closed.Load() {
    c.c <- val
  }
  c.mtx.RUnlock()
}

func (c *Channel[T]) Close() {
  if !c.closed.Swap(true) {
    c.mtx.Lock()
    close(c.c)
    c.mtx.Unlock()
  }
}

var (
  // map[UUID]*webs.Conn
  // map[UUID]chan Message
  clients sync.Map
)

func main() {
  log.SetFlags(log.Lshortfile)
  if len(os.Args) != 2 {
    log.Fatal("must provide the address (and only the address)")
  }
  addr := os.Args[1]
  http.Handle("/", webs.Handler(handler))
  log.Printf("Listening on %s", addr)
  log.Fatal(http.ListenAndServe(addr, nil))
}

func handler(ws *webs.Conn) {
  defer ws.Close()
  uuid := uuidpkg.New().String()
  logFunc := func(format string, args ...any) {
    log.Output(
      2,
      fmt.Sprintf(
        fmt.Sprintf("[%s|%s] %s", ws.Request().RemoteAddr, uuid, format),
        args...,
      ),
    )
  }

  msg := common.NewSystemMessage(common.ActionConnect, uuid)
  msgJSONBytes, err := json.Marshal(msg)
  if err != nil {
    webs.JSON.Send(ws, common.NewSystemMessage(common.ActionError, "internal server error"))
    logFunc("error marshaling json: %v", err)
    return
  }
  // Don't add ws to clients until after sending connect so that messages
  // aren't received before the connect is sent to all.
  broadcastMsgBytes(msgJSONBytes)
  ws.Write(msgJSONBytes)
  channel := NewChannel[[]byte](50)
  //clients.Store(uuid, ws)
  clients.Store(uuid, channel)

  go func() {
    for msg := range channel.c {
      ws.Write(msg)
    }
  }()

  defer func() {
    //clients.Delete(uuid)
    msg := common.NewSystemMessage(common.ActionDisconnect, uuid)
    /*
    if msgJSONBytes, err := json.Marshal(msg); err == nil {
      ws.Write(msgJSONBytes)
      broadcastMsgBytes(msgJSONBytes)
    }
    */
    go broadcastMsg(msg)
    channel.Close()
    clients.Delete(uuid)
  }()

  unmarshalTypeError := &json.UnmarshalTypeError{}
  for {
    if err := webs.JSON.Receive(ws, &msg); err != nil {
      if errors.As(err, &unmarshalTypeError) {
        webs.JSON.Send(ws, common.NewSystemMessage(common.ActionError, "bad message"))
      } else if err != io.EOF {
        logFunc("error reading from client: %v", err)
      }
      return
    }
    go broadcastMsg(common.NewChatMessage(uuid, msg.Contents))
  }
}

func broadcastMsg(msg common.Message) error {
  msgJSONBytes, err := json.Marshal(msg)
  if err != nil {
    return err
  }
  broadcastMsgBytes(msgJSONBytes)
  return nil
}

func broadcastMsgBytes(b []byte) {
  //clients.Range(func(_, iWs any) bool {
    //iWs.(*webs.Conn).Write(b)
  clients.Range(func(_, iChannel any) bool {
    iChannel.(*Channel[[]byte]).Send(b)
    return true
  })
}
