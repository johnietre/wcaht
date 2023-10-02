#pragma once

/*
#include "message.hpp"
#include "include/listener.hpp"
#include "include/server.hpp"
#include "include/http_session.hpp"
#include "include/websocket.hpp"
#include <iostream>
*/

void die(const char *msg) {
  std::cerr << msg << std::endl;
  std::exit(1);
}
