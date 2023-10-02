#pragma once

#include "../message.hpp"
#include <boost/smart_ptr.hpp>
#include <memory>
#include <mutex>
#include <shared_mutex>
#include <string>
#include <unordered_set>

using namespace message;

class WsConn;

class Server : public boost::enable_shared_from_this<Server> {
  mutable std::shared_mutex mtx_;
  //std::shared_mutex mtx_;
  //std::mutex mtx_;
  std::unordered_set<WsConn*> clients_;

public:
  explicit Server();

  void connect(WsConn*);

  void disconnect(WsConn*);

  void broadcast_msg(Message);

  void broadcast_msg_text(boost::shared_ptr<std::string const>);
};
