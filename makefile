bin:
	-@mkdir bin

bin/client: bin ./client/*.go ./wschat-go/common/message.go
	go build -o $@ ./client

bin/server: bin ./server/*.go
	go build -o $@ ./server

bin/wschat-go: bin ./wschat-go/*.go
	go build -o $@ ./wschat-go

bin/wschat-py: bin ./wschat-py/run.sh
	SCRIPT_PATH="$(shell pwd)/wschat-py/run.sh" DIR="./wschat-py" rustc ./runner/main.rs -o $@

bin/wschat-rust: bin ./wschat-rust/src/*.rs
	cargo build -r --manifest-path=./wschat-rust/Cargo.toml
	cp wschat-rust/target/release/wschat-rust ./bin

### C++ ###
# TODO: Add .hpp files to dependencies

export BOOST_PATH ?= wschat-cpp/libs/boost_1_82_0

wschat-cpp/include/output/http_session.o: wschat-cpp/include/http_session.cpp
	clang++ $< --std=c++17 -c -o $@ -I$(BOOST_PATH) -Ofast

wschat-cpp/include/output/listener.o: wschat-cpp/include/listener.cpp
	clang++ $< --std=c++17 -c -o $@ -I$(BOOST_PATH) -Ofast

wschat-cpp/include/output/server.o: wschat-cpp/include/server.cpp
	clang++ $< --std=c++17 -c -o $@ -I$(BOOST_PATH) -Ofast

wschat-cpp/include/output/websocket.o: wschat-cpp/include/websocket.cpp wschat-cpp/include/server.hpp
	clang++ $< --std=c++17 -c -o $@ -I$(BOOST_PATH) -Ofast

wschat-cpp/output/main.o: wschat-cpp/main.cpp
	clang++ $< --std=c++17 -c -o $@ -I$(BOOST_PATH) -Ofast

CPP_OBJS = wschat-cpp/include/output/$(wildcard *.o) wschat-cpp/output/main.o

#bin/wschat-cpp: $(CPP_OBJS)
bin/wschat-cpp: wschat-cpp/include/output/http_session.o wschat-cpp/include/output/listener.o wschat-cpp/include/output/server.o wschat-cpp/include/output/websocket.o wschat-cpp/main.cpp
	clang++ $^ --std=c++17 -o $@ -I$(BOOST_PATH) -Ofast

### Commands to build ###

client: bin/client
server: bin/server
wschat-cpp: bin/wschat-cpp
wschat-go: bin/wschat-go
wschat-py: bin/wschat-py
wschat-rust: bin/wschat-rust
