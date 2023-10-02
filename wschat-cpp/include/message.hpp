#pragma once

#include <string>
#include <string_view>

namespace message {

enum class Action {
  Connect,
  Chat,
  Disconnect,
  Error,
};

class InvalidActionError;

class JSONError;

struct Message {
  std::string sender;
  Action action;
  std::string contents;
  int64_t timestamp;

  Message();
  Message(std::string_view);
  Message(Action, std::string contents);

  void deserialize(std::string_view);
  std::string serialize() const;
};

};
