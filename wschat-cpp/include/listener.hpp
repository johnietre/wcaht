#pragma once

#include "beast.hpp"
#include "net.hpp"
#include <boost/smart_ptr.hpp>
#include <memory>
#include <string>

class Server;

class Listener : public boost::enable_shared_from_this<Listener> {
  net::io_context& ioc_;
  tcp::acceptor acceptor_;
  boost::shared_ptr<Server> server_;

  void fail(beast::error_code, char const*);
  void accept(beast::error_code, tcp::socket);
public:
  Listener(net::io_context&, tcp::endpoint, boost::shared_ptr<Server> const& server);
  void run();
};
