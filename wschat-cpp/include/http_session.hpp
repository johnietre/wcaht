#pragma once

#include "net.hpp"
#include "beast.hpp"
#include "server.hpp"
#include <boost/optional.hpp>
#include <boost/smart_ptr.hpp>
#include <cstdlib>
#include <memory>

class Server;

class HttpSession: public boost::enable_shared_from_this<HttpSession> {
  beast::tcp_stream stream_;
  beast::flat_buffer buffer_;
  boost::shared_ptr<Server> server_;
  // The parser is stored as optional so we can construct it from scratch at
  // beginning of each new message.
  boost::optional<http::request_parser<http::string_body>> parser_;

  struct send_lambda;

  void fail(beast::error_code, char const*);
  void do_read();
  void handle(beast::error_code, std::size_t);
  //void on_read(beast::error_code, std::size_t);
  void on_write(beast::error_code, std::size_t, bool);

  //template<class Body, class Allocator>
  //static http::message_generator handle_request(http::request<Body, http::basic_fields<Allocator>>&&);

public:
  HttpSession(tcp::socket&&, boost::shared_ptr<Server> const&);

  void run();
};
