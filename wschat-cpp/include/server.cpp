#include "../message.hpp"
#include "server.hpp"
#include "websocket.hpp"
#include <boost/smart_ptr.hpp>
#include <mutex>
#include <shared_mutex>
#include <string>

using namespace message;

/******************** Server ********************/

Server::Server() {}

void Server::connect(WsConn* ws) {
  auto msg_text = Message(Action::Connect, ws->uuid).serialize();
  auto ss = boost::make_shared<std::string const>(std::move(msg_text));
  ws->send(ss);
  this->broadcast_msg_text(ss);
  std::lock_guard lock(this->mtx_);
  this->clients_.insert(ws);
}

void Server::disconnect(WsConn* ws) {
  this->broadcast_msg(Message(Action::Disconnect, ws->uuid));
  std::lock_guard lock(this->mtx_);
  this->clients_.erase(ws);
}

void Server::broadcast_msg(Message msg) {
  auto msg_text = msg.serialize();
  this->broadcast_msg_text(boost::make_shared<std::string const>(std::move(msg_text)));
}

void Server::broadcast_msg_text(boost::shared_ptr<std::string const> ss) {
  std::shared_lock lock(this->mtx_);
  //std::lock_guard lock(this->mtx_);
  for (auto ws : this->clients_) {
    auto weak = ws->weak_from_this();
    if (auto strong = weak.lock()) {
      strong->send(ss);
    }
  }
}
