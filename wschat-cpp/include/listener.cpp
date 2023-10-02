#include "listener.hpp"
#include "http_session.hpp"
#include <iostream>

Listener::Listener(net::io_context& ioc, tcp::endpoint endpoint, boost::shared_ptr<Server> const& server) : ioc_(ioc), acceptor_(ioc), server_(server) {
  beast::error_code ec;

  this->acceptor_.open(endpoint.protocol(), ec);
  if (ec) {
    this->fail(ec, "open");
    return;
  }

  this->acceptor_.set_option(net::socket_base::reuse_address(true), ec);
  if (ec) {
    this->fail(ec, "set_option");
    return;
  }

  this->acceptor_.bind(endpoint, ec);
  if (ec) {
    this->fail(ec, "bind");
    return;
  }

  this->acceptor_.listen(net::socket_base::max_listen_connections, ec);
  if (ec) {
    this->fail(ec, "listen");
    return;
  }
}

void Listener::fail(beast::error_code ec, char const* what) {
  if (ec != net::error::operation_aborted) {
    std::cerr << what << ": " << ec.message() << '\n';
  }
}

void Listener::accept(beast::error_code ec, tcp::socket socket) {
  if (ec) {
    this->fail(ec, "accept");
    return;
  }
  boost::make_shared<HttpSession>(std::move(socket), this->server_)->run();
  this->acceptor_.async_accept(
    net::make_strand(this->ioc_),
    beast::bind_front_handler(&Listener::accept, this->shared_from_this())
  );
}

void Listener::run() {
  this->acceptor_.async_accept(
    net::make_strand(this->ioc_),
    beast::bind_front_handler(&Listener::accept, this->shared_from_this())
  );
}
