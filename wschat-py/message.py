from __future__ import annotations
import json, time
from typing import NewType

Action = NewType("Action", str)

ACTION_CONNECT = Action("connect")
ACTION_CHAT = Action("chat")
ACTION_DISCONNECT = Action("disconnect")
ACTION_ERROR = Action("error")

def action_is_valid(action: str):
    return action in set([ACTION_CONNECT, ACTION_CHAT, ACTION_DISCONNECT, ACTION_ERROR])

class Message:
    def __init__(
            self, sender: str = "", action: Action = "",
            contents: str = "", timestamp: int = 0,
        ):
        #if not action_is_valid(action):
            #raise ValueError(f"invalid action: {action}")
        self.sender = sender
        #self._action = action
        self.action = action
        self.contents = contents
        self.timestamp = timestamp

    @staticmethod
    def new_system(action: str, contents: str) -> Message:
        return Message("system", action, contents, time.time_ns())

    @staticmethod
    def new_chat(sender: str, contents: str) -> Message:
        return Message(sender, "chat", contents, time.time_ns())

    def get_action(self) -> Action:
        return self._action

    def set_action(self, action: str):
        if not action_is_valid(action):
            raise ValueError(f"invalid action: {action}")
        self._action = Action(action)

    def del_action(self):
        del self._action

    action = property(get_action, set_action, del_action)

    def to_json(self):
        return json.dumps({
            "sender": self.sender,
            "action": self.action,
            "contents": self.contents,
            "timestamp": self.timestamp,
        })

    @staticmethod
    def from_json(s) -> Message:
        d = json.loads(s)
        # TODO: Throw JSONDecodeError?
        sender = d.get("sender")
        if sender is not None and type(sender) != str:
            raise ValueError(f"expected 'str' sender, got {type(sender)}")

        action = d.get("action")
        if action is not None and type(action) != str:
            raise ValueError(f"expected 'str' action, got {type(action)}")

        contents = d.get("contents")
        if contents is not None and type(contents) != str:
            raise ValueError(f"expected 'str' contents, got {type(contents)}")

        timestamp = d.get("timestamp")
        if timestamp is not None and type(timestamp) != int:
            raise ValueError(f"expected 'int' timestamp, got {type(timestamp)}")

        return Message(sender, action, contents, timestamp)
