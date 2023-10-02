#include "beast.hpp"
#include "net.hpp"
#include "websocket.hpp"
#include <boost/uuid/uuid.hpp>
#include <boost/uuid/uuid_generators.hpp>
#include <boost/uuid/uuid_io.hpp>
#include <iostream>

void WsConn::fail(beast::error_code ec, char const* what) {
  if (ec != net::error::operation_aborted && ec != websocket::error::closed) {
    //std::cerr << what << ": " << ec.message() << '\n';
  }
}

void WsConn::accept(beast::error_code ec) {
  if (ec) {
    this->fail(ec, "accept");
    return;
  }
  this->server_->connect(this);
  this->ws_.async_read(this->buffer_, beast::bind_front_handler(
    &WsConn::handle, this->shared_from_this()
  ));
}

void WsConn::handle(beast::error_code ec, size_t bytes_transferred) {
  if (ec) {
    this->fail(ec, "handle");
    return;
  }
  auto json_msg = beast::buffers_to_string(this->buffer_.data());
  try {
    Message msg = Message(json_msg);
    this->server_->broadcast_msg(msg);
  } catch (std::exception &e) {
    Message msg = Message(Action::Error, "bad message");
    json_msg = msg.serialize();
    this->send(boost::make_shared<std::string const>(json_msg));
  }
  this->buffer_.consume(this->buffer_.size());
  this->ws_.async_read(this->buffer_, beast::bind_front_handler(
    &WsConn::handle, this->shared_from_this()
  ));
}

void WsConn::on_write(beast::error_code ec, std::size_t bytes_transferred) {
  if (ec) {
    this->fail(ec, "write");
    return;
  }
  this->queue_.pop();

  if (!this->queue_.empty()) {
    this->ws_.async_write(net::buffer(*this->queue_.front()), beast::bind_front_handler(
      &WsConn::on_write, this->shared_from_this()
    ));
  }
}

void WsConn::on_send(boost::shared_ptr<std::string const> const& ss) {
  this->queue_.push(ss);
  // Check if we are already writing
  if (this->queue_.size() > 1) {
    return;
  }
  // If not, send immediately
  this->ws_.async_write(net::buffer(*this->queue_.front()), beast::bind_front_handler(
    &WsConn::on_write, this->shared_from_this()
  ));
}

WsConn::WsConn(tcp::socket&& socket, boost::shared_ptr<Server> const& server)
  : ws_(std::move(socket)), server_(server), uuid(boost::uuids::to_string(boost::uuids::random_generator()())) {}

WsConn::~WsConn() {
  // Send disconnect message
  this->server_->disconnect(this);
}

void WsConn::send(boost::shared_ptr<std::string const> const& ss) {
  // Post to the strand to ensure the members of `this` aren't accessed
  // concurrently.
  net::post(this->ws_.get_executor(), beast::bind_front_handler(
    &WsConn::on_send, this->shared_from_this(), ss
  ));
}
