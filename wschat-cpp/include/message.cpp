#include "message.hpp"
#include "../libs/json/single_include/nlohmann/json.hpp" // json
#include <chrono>      // nanoseconds, system_clock, time_point_cast
#include <exception>   // exception
#include <stdexcept>   // runtime_error
#include <string>      // string, to_string
#include <string_view> // string_view

namespace message {

using json = nlohmann::json;

namespace chrono = std::chrono;

int64_t get_timestamp() {
  return chrono::time_point_cast<chrono::nanoseconds>(
             chrono::system_clock::now())
      .time_since_epoch()
      .count();
}

class InvalidActionError : public std::runtime_error {
public:
  InvalidActionError(const std::string &err) : std::runtime_error(err) {}
};

class JSONError : public std::runtime_error {
public:
  JSONError(const std::string &err) : std::runtime_error(err) {}
  JSONError(const std::exception &err) : std::runtime_error(err.what()) {}
};

const size_t MAX_ACTION_STR_LEN = 10;

const char *action_str(Action action) {
  switch (action) {
  case Action::Connect:
    return "connect";
  case Action::Chat:
    return "chat";
  case Action::Disconnect:
    return "disconnect";
  case Action::Error:
    return "error";
  default:
    throw InvalidActionError("shouldn't be reached");
  }
}

Action action_from_string(std::string_view sv) {
  if (sv == "connect") {
    return Action::Connect;
  } else if (sv == "chat") {
    return Action::Chat;
  } else if (sv == "disconnect") {
    return Action::Disconnect;
  } else if (sv == "error") {
    return Action::Error;
  }
  throw InvalidActionError("invalid action: " + std::string(sv));
}

void to_json(json &j, const Message &msg) {
  j = json{{"sender", msg.sender},
           {"action", std::string(action_str(msg.action))},
           {"contents", msg.contents},
           {"timestamp", msg.timestamp}};
}

void from_json(const json &j, Message &msg) {
  j.at("sender").get_to(msg.sender);
  if (j.contains("action")) {
    msg.action = action_from_string(j.at("action").get<std::string>());
  } else {
    msg.action = Action::Chat;
  }
  if (j.contains("contents")) {
    j.at("contents").get_to(msg.contents);
  }
  if (j.contains("timestamp")) {
    j.at("timestamp").get_to(msg.timestamp);
  }
}

Message::Message() : sender(""), action(Action::Error), contents(""), timestamp(0) {}
  // Deserialize a JSON string
Message::Message(std::string_view sv) { this->deserialize(sv); }
Message::Message(Action action, std::string contents)
      : sender("system"), action(action), contents(contents),
        timestamp(get_timestamp()) {}
Message::Message(std::string sender, std::string contents)
      : sender(sender), action(Action::Chat), contents(contents),
        timestamp(get_timestamp()) {}

void Message::deserialize(std::string_view sv) {
  try {
    auto msg_json = json::parse(sv);
    msg_json.get_to(*this);
  } catch (std::exception &e) {
    throw JSONError(e);
  }
}

std::string Message::serialize() const {
  try {
    // json msg_json = this;
    json msg_json;
    to_json(msg_json, *this);
    return msg_json.dump();
  } catch (const std::exception &e) {
    throw JSONError(e);
  }
}

  /*
  void deserialize(std::string_view sv) {
    size_t len = sv.length();
    if (len == 0) {
      return;
    }
    if (sv[0] != '{') {
      throw TODO;
    }
    bool reading_field_name = false;
    // LSB is sender, next is action, contents, then timestamp
    // MSB means an unknown field
    // Non-zero value means a field is being read
    uint_8 fields = 0;
    char prev, expected_next = sv[0], '\0';
    size_t start = 2, i == 2;
    bool finished = false;
    for (; i < sv.length(); i++) {
      char c = sv[i];
      // Not getting a field name nor reading a value
      if (std::isspace(c) && !reading_field_name && fields == 0) {
        continue;
      }
      if (expected_next != '\0') {
        if (expected_next != c) {
          throw TODO;
        }
        expected_next = '\0';
      }
      if (prev == '\\') {
        if (c != '\\' && c != '"') {
          throw TODO;
        }
        if (reading_field_name) {
          // Do nothing right now
        } else if (fields != 0) {
          // The current char can be added on the next append
          switch (fields) {
            case 1:
              sender.append(sv.substr(start, i - start - 1));
              break;
            case 2:
              // Go ahead and throw since this is will be an invalid action
              throw TODO;
            case 4:
              contents.append(sv.substr(start, i - start - 1));
              break;
            case 8:
              // Go ahead and throw since this will be an invalid timestamp
              throw TODO;
              break;
            default:
              break;
          }
          start = i;
        }
        prev = '\0';
        continue;
      }
      if (c == '"") {
        if (reading_field_name) {
          // Don't worry about backslashes/quotes in the name right now
          std::string_view name = sv.substr(start, i - start);
          if (name == "contents") {
            fields = 1;
          } else if (name == "sender") {
            fields = 2;
          } else if (name == "action") {
            fields = 4;
          } else if (name == "timestamp") {
            fields = 8;
          } else {
            fields = 128;
          }
          expected_next = ':';
          start = i; // TODO
        } else if (fields != 0) {
          //
        } else if (prev == ',' || prev == '{') {
          reading_field_name = true;
        } else if (prev != ':') {
          //
        }
      }
      prev = c;
    }
    if (!finished) {
      throw TODO;
    }
  }

  void serialize_into(std::string &s) const {
    auto const action_str = message::action_str(this->action);
    auto const expected =
      1 + 9 + 1 + this->sender.length() + 1 // {"sender":"this->sender"
      + 1 + 9 + 1 + MAX_ACTION_STR_LEN + 1 // ,"action":"this->action"
      + 1 + 11 + 1 + this->contents.length() + 1 // ,"contents":"this->contents"
      + 1 + 12 + 20 + 1; // ,"timestamp":this->timestamp}
    if (s.capacity() - s.length() < expected) {
      s.reserve(s.capacity() + (expected - (s.capacity() - s.length())));
    }
    s.append("{\"sender\":\"");
    s.append(this->sender);
    s.append("\",\"action\":\"");
    s.append(action_str);
    s.append("\",\"contents\":\"");
    size_t start = 0, i = 0, contents_len = this->contents.length();
    for (; i < contents_len; i++) {
      char const c = this->contents[i];
      if (c == '\\' || c == '"') {
        s.append(this->contents.substr(start, i - start));
        s.push_back('\\');
        // The current char can be added on the next append
        start = i;
      }
    }
    if (start != i) {
      s.append(this->contents.substr(start));
    }
    //for (const auto c : this->contents) {
      //if (c == '\\' || c == '"') {
        //s.push_back('\\')
      //}
      //s.push_back(
    //}
    s.append("\",\"timestamp\":");
    s.append(std::to_string(this->timestamp));
    s.push_back('}');

    // ,"timestamp":this->timestamp}
    //if (s.capacity() - s.length() < 34) {
      //s.reserve(s.capacity() + (34 - (s.capacity() - s.length())));
    //}
    //std::sprintf(s.data(), "\"timestamp\":%lld", this->timestamp);
    // Replace the null terminator placed by std::sprintf
    //s[s.length() - 1] = '}';
  }

  std::string serialize() const {
    std::string s;
    serialize_into(s);
    return std::move(s);
  }
  */
};

} // namespace message
