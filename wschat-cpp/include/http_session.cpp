#include "http_session.hpp"
#include "websocket.hpp"
#include <boost/config.hpp>
#include <iostream>

/******************** HttpSession ********************/

template<class Body, class Allocator>
//http::message_generator HttpSession::handle_request(http::request<Body, http::basic_fields<Allocator>>&& req) {
http::message_generator handle_request(http::request<Body, http::basic_fields<Allocator>>&& req) {
  // For right now, send not found for anything that isn't the base path.

  if (req.target().empty() || req.target()[0] == '/') {
    http::response<http::string_body> res{http::status::bad_request, req.version()};
    //res.set(http::field::server, BOOST_BEAST_VERSION_STRING);
    res.set(http::field::content_type, "text/html");
    res.keep_alive(req.keep_alive());
    res.body() = "not websocket protocol";
    res.prepare_payload();
    return res;
  }

  http::response<http::string_body> res{http::status::not_found, req.version()};
  //res.set(http::field::server, BOOST_BEAST_VERSION_STRING);
  res.set(http::field::content_type, "text/html");
  res.keep_alive(req.keep_alive());
  //res.body() = "The resource '" + std::string(target) + "' was not found.";
  res.prepare_payload();
  return res;
}

void HttpSession::fail(beast::error_code ec, char const* what) {
  if (ec != net::error::operation_aborted) {
    std::cerr << what << ": " << ec.message() << '\n';
  }
}

void HttpSession::do_read() {
  // Construct a new parser for each message
  this->parser_.emplace();
  // Apply a reasonable limit to the allowed size of the body to prevent
  // abuse.
  //this->parser_->body_limit(10000);
  //this->stream_.expires_after(std::chrono::seconds(30));

  http::async_read(
    this->stream_, this->buffer_, this->parser_->get(),
    beast::bind_front_handler(
      &HttpSession::handle, this->shared_from_this()
      //&HttpSession::on_read, this->shared_from_this()
    )
  );
}

void HttpSession::handle(beast::error_code ec, std::size_t) {
//void HttpSession::on_read(beast::error_code ec, std::size_t) {
  if (ec) {
    if (ec == http::error::end_of_stream) {
      this->stream_.socket().shutdown(tcp::socket::shutdown_send, ec);
    } else {
      this->fail(ec, "read");
    }
    return;
  }

  if (websocket::is_upgrade(this->parser_->get())) {
    boost::make_shared<WsConn>(
      this->stream_.release_socket(), this->server_
    )->run(this->parser_->release());
    return;
  }

  // Handle request
  http::message_generator msg = handle_request(this->parser_->release());
  bool keep_alive = msg.keep_alive();
  auto self = this->shared_from_this();

  // Send the response
  beast::async_write(
    this->stream_, std::move(msg),
    [self, keep_alive](beast::error_code ec, std::size_t bytes) {
      self->on_write(ec, bytes, keep_alive);
    }
  );
}

void HttpSession::on_write(beast::error_code ec, std::size_t, bool keep_alive) {
  if (ec) {
    this->fail(ec, "write");
    return;
  }
  if (!keep_alive) {
    this->stream_.socket().shutdown(tcp::socket::shutdown_send, ec);
    return;
  }
  // Read another request
  this->do_read();
}

HttpSession::HttpSession(tcp::socket&& socket, boost::shared_ptr<Server> const& server)
  : stream_(std::move(socket)), server_(server) {
  }

void HttpSession::run() {
  this->do_read();
}
