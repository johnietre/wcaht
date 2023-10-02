#include "include/listener.hpp"
#include "include/server.hpp"
#include <boost/asio/signal_set.hpp>
#include <boost/smart_ptr.hpp>
#include <iostream>
#include <string_view>
#include <vector>

void die(const char *msg) {
  std::cerr << msg << std::endl;
  std::exit(1);
}

int main(int argc, char **argv) {
  if (argc == 1) {
    die("must provide address");
  }

  auto const addr = std::string_view(argv[1]);
  auto const pos = addr.rfind(":");
  if (pos == std::string_view::npos) {
    die("must provide valid address");
  }

  auto ip = net::ip::make_address(std::string(addr.substr(0, pos)));
  unsigned short port;
  try {
    port = static_cast<unsigned short>(
        std::stoul(std::string(addr.substr(pos + 1))));
  } catch (...) {
    die("must provide valid address");
  }

  auto const workers = (argc > 2) ? std::max<int>(1, std::atoi(argv[2])) : 1;

  net::io_context ioc;

  boost::make_shared<Listener>(ioc, tcp::endpoint{ip, port},
                               boost::make_shared<Server>())
      ->run();

  net::signal_set signals(ioc, SIGINT, SIGTERM);
  signals.async_wait(
      [&ioc](boost::system::error_code const &, int) { ioc.stop(); });

  std::cout << "Listening on " << addr << '\n';
  std::vector<std::thread> threads;
  threads.reserve(workers - 1);
  for (auto i = workers - 1; i > 0; --i) {
    threads.emplace_back([&ioc] { ioc.run(); });
  }
  ioc.run();

  for (auto &t : threads) {
    t.join();
  }

  return 0;
}
