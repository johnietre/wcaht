package common

import (
  "encoding/json"
  "fmt"
  "time"
)

type Message struct {
  Sender string `json:"sender,omitempty"`
  Action Action `json:"action,omitempty"`
  Contents string `json:"contents,omitempty"`
  Timestamp int64 `json:"timestamp,omitempty"`
}

func NewSystemMessage(action Action, contents string) Message {
  return Message{
    Sender: "system",
    Action: action,
    Contents: contents,
    Timestamp: time.Now().UnixNano(),
  }
}

func NewChatMessage(sender, contents string) Message {
  return Message{
    Sender: sender,
    Action: ActionChat,
    Contents: contents,
    Timestamp: time.Now().UnixNano(),
  }
}

type Action string

const (
  ActionConnect Action = "connect"
  ActionChat = "chat"
  ActionDisconnect = "disconnect"
  ActionError = "error"
)

func (a Action) MarshalJSON() ([]byte, error) {
  switch a {
  case ActionConnect, ActionChat, ActionDisconnect, ActionError:
    return json.Marshal(string(a))
  }
  return nil, fmt.Errorf("invalid action: %s", a)
}

func (a *Action) UnmarshalJSON(b []byte) error {
  str := ""
  if err := json.Unmarshal(b, &str); err != nil {
    return err
  }
  action := Action(str)
  switch action {
  case ActionConnect, ActionChat, ActionDisconnect, ActionError:
    *a = action
    return nil
  }
  return fmt.Errorf("invalid action: %s", str)
}
