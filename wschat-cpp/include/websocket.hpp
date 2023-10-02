#pragma once

#include "net.hpp"
#include "beast.hpp"
#include "server.hpp"

#include <cstdlib>
#include <memory>
#include <queue>
#include <string>
#include <vector>

class Server;

class WsConn : public boost::enable_shared_from_this<WsConn> {
  beast::flat_buffer buffer_;
  websocket::stream<beast::tcp_stream> ws_;
  boost::shared_ptr<Server> server_;
  std::queue<boost::shared_ptr<std::string const>> queue_;

  void fail(beast::error_code, char const*);
  void accept(beast::error_code);
  void handle(beast::error_code, std::size_t);
  void on_write(beast::error_code, std::size_t);
  void on_send(boost::shared_ptr<std::string const> const&);

protected:
  std::string uuid;

public:
  friend class Server;

  WsConn(tcp::socket&&, boost::shared_ptr<Server> const&);
  ~WsConn();

  template<class Body, class Allocator>
  void run(http::request<Body, http::basic_fields<Allocator>>);
  void send(boost::shared_ptr<std::string const> const&);
};

template<class Body, class Allocator>
void WsConn::run(http::request<Body, http::basic_fields<Allocator>> req) {
  /*
  this->ws_.set_option(websocket::stream_base::timeout::suggested(
    beast::role_type::server
  ));
  */

  // Set a decorator to change the Server of the handshake
  /*
  this->ws_.set_option(websocket::stream_base::decorator(
    [](websocket::response_type& res) {
      res.set(http::field::server, std::string(BOOST_BEAST_VERSION_STRING) +
        " wschat-cpp");
    }
  ));
  */

  // Accept the websocket handshake
  this->ws_.async_accept(req, beast::bind_front_handler(
      &WsConn::accept, this->shared_from_this()
    ));
}
