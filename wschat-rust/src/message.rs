use serde::de::{self, Deserialize, Deserializer, Visitor};
use serde::ser::{Serialize, Serializer};
use std::fmt;
use std::time::SystemTime;

pub const fn action_chat() -> Action {
    Action::Chat
}

#[derive(actix::Message, serde::Serialize, serde::Deserialize)]
#[rtype(result = "()")]
pub struct Message {
    pub sender: String,
    #[serde(default = "action_chat")]
    pub action: Action,
    #[serde(default)]
    pub contents: String,
    #[serde(default)]
    pub timestamp: u64,
}

impl Message {
    pub fn new_system(action: Action, contents: impl ToString) -> Self {
        Self {
            sender: String::from("system"),
            action,
            contents: contents.to_string(),
            timestamp: get_timestamp(),
        }
    }

    pub fn new_chat(sender: String, contents: impl ToString) -> Self {
        Self {
            sender,
            action: Action::Chat,
            contents: contents.to_string(),
            timestamp: get_timestamp(),
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Action {
    Connect,
    Chat,
    Disconnect,
    Error,
}

impl Action {
    const fn as_str(self) -> &'static str {
        match self {
            Action::Connect => "connect",
            Action::Chat => "chat",
            Action::Disconnect => "discconnect",
            Action::Error => "error",
        }
    }
}

impl std::str::FromStr for Action {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "connect" => Ok(Action::Connect),
            "chat" => Ok(Action::Chat),
            "error" => Ok(Action::Error),
            "disconnect" => Ok(Action::Disconnect),
            _ => Err(format!(r"invalid value: {}", s)),
        }
    }
}

impl Serialize for Action {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(self.as_str())
    }
}

impl<'de> Deserialize<'de> for Action {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        deserializer.deserialize_str(ActionVisitor)
    }
}

struct ActionVisitor;

impl<'de> Visitor<'de> for ActionVisitor {
    type Value = Action;

    fn expecting(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(
            f,
            r#"one of the following: "connect", "chat", "disconnect", "error""#
        )
    }

    fn visit_str<E>(self, v: &str) -> Result<Self::Value, E>
    where
        E: de::Error,
    {
        v.parse().map_err(E::custom)
    }
}

pub fn get_timestamp() -> u64 {
    SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .unwrap()
        .as_nanos() as _
}
