// TODO: Stop cloning UUID field?
use actix::prelude::{
    Actor, ActorContext, Addr, AsyncContext, Context, Handler, Recipient, Running, StreamHandler,
};
use actix_web::{web, App, Error, HttpRequest, HttpResponse, HttpServer};
use actix_web_actors::ws;
//use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;

mod message;
use message::{Action, Message};

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let args = std::env::args().skip(1).collect::<Vec<_>>();
    if args.len() == 0 {
        die("must provide address");
    }

    env_logger::init();

    let addr = &args[0];
    let workers = args
        .get(1)
        .map(|n| n.parse().unwrap_or_else(|e| die(&format!("{}", e))))
        .unwrap_or(1);
    let server = Server::new().start();
    println!("Listening on {}", addr);
    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(server.clone()))
            .route("/", web::get().to(handler))
    })
    .workers(workers)
    .bind(addr)
    .unwrap()
    .run()
    .await
}

fn die(msg: &str) -> ! {
    eprintln!("{}", msg);
    std::process::exit(1)
}

async fn handler(
    req: HttpRequest,
    stream: web::Payload,
    server_addr: web::Data<Addr<Server>>,
) -> Result<HttpResponse, Error> {
    ws::start(Conn::new(server_addr.get_ref().clone()), &req, stream)
}

struct Conn {
    uuid: String,
    server_addr: Addr<Server>,
}

impl Conn {
    fn new(server_addr: Addr<Server>) -> Self {
        Self {
            uuid: uuid::Uuid::new_v4().to_string(),
            server_addr,
        }
    }
}

impl Actor for Conn {
    type Context = ws::WebsocketContext<Self>;

    fn started(&mut self, ctx: &mut Self::Context) {
        self.server_addr.do_send(Connect {
            uuid: self.uuid.clone(),
            recip: ctx.address().recipient(),
        });
    }

    fn stopping(&mut self, _: &mut Self::Context) -> Running {
        self.server_addr.do_send(Disconnect(self.uuid.clone()));
        Running::Stop
    }
}

impl StreamHandler<Result<ws::Message, ws::ProtocolError>> for Conn {
    fn handle(&mut self, msg: Result<ws::Message, ws::ProtocolError>, ctx: &mut Self::Context) {
        let msg = match msg {
            Ok(ws::Message::Ping(msg)) => {
                ctx.pong(&msg);
                return;
            }
            Ok(ws::Message::Pong(_)) => return,
            Ok(ws::Message::Text(text)) => text.to_string(),
            // Handle anyway?
            Ok(ws::Message::Binary(_)) => return,
            Ok(ws::Message::Close(reason)) => {
                ctx.close(reason);
                ctx.stop();
                return;
            }
            Ok(ws::Message::Continuation(_)) => {
                ctx.stop();
                return;
            }
            Ok(ws::Message::Nop) => return,
            Err(_) => {
                // TODO: Log
                ctx.stop();
                return;
            }
        };
        let msg: Message = match serde_json::from_str(&msg) {
            Ok(msg) => msg,
            Err(e) => {
                log::info!("bad message: {}", e);
                ctx.text(
                    serde_json::to_string(&Message::new_system(Action::Error, "bad message"))
                        .unwrap(),
                );
                return;
            }
        };
        self.server_addr
            .do_send(Message::new_chat(self.uuid.clone(), msg.contents));
    }
}

impl Handler<SerMsg> for Conn {
    type Result = ();

    fn handle(&mut self, msg: SerMsg, ctx: &mut Self::Context) {
        ctx.text(&*msg.0)
    }
}

struct Server {
    //clients: HashMap<String, Recipient<Message>>,
    clients: HashMap<String, Recipient<SerMsg>>,
}

impl Server {
    fn new() -> Self {
        Self {
            clients: HashMap::new(),
        }
    }

    fn broadcast_msg(&self, msg: Message) {
        match serde_json::to_string(&msg) {
            Ok(s) => self.broadcast_msg_text(SerMsg::from(s)),
            Err(_) => return,
        }
    }

    fn broadcast_msg_text(&self, msg: SerMsg) {
        self.clients
            .values()
            .for_each(|recip| recip.do_send(msg.clone()));
    }
}

impl Actor for Server {
    type Context = Context<Self>;
}

impl Handler<Message> for Server {
    type Result = ();

    fn handle(&mut self, msg: Message, _: &mut Self::Context) {
        self.broadcast_msg(msg);
    }
}

impl Handler<Connect> for Server {
    type Result = ();

    fn handle(&mut self, msg: Connect, _: &mut Self::Context) {
        let ser_msg = SerMsg::from(
            serde_json::to_string(&Message::new_system(Action::Connect, msg.uuid.clone())).unwrap(),
        );
        self.broadcast_msg_text(ser_msg.clone());
        msg.recip.do_send(ser_msg);
        self.clients.insert(msg.uuid, msg.recip);
    }
}

impl Handler<Disconnect> for Server {
    type Result = ();

    fn handle(&mut self, msg: Disconnect, _: &mut Self::Context) {
        self.broadcast_msg(Message::new_system(Action::Disconnect, msg.0.clone()));
        self.clients.remove(&msg.0);
    }
}

#[derive(actix::Message)]
#[rtype(result = "()")]
struct SerMsg(Arc<str>);

impl From<String> for SerMsg {
    fn from(s: String) -> Self {
        Self(Arc::from(s))
    }
}

impl Clone for SerMsg {
    fn clone(&self) -> Self {
        Self(Arc::clone(&self.0))
    }
}

#[derive(actix::Message)]
#[rtype(result = "()")]
struct Connect {
    uuid: String,
    //recip: Recipient<Arc<str>>,
    recip: Recipient<SerMsg>,
}

// Holds uuid
#[derive(actix::Message)]
#[rtype(result = "()")]
struct Disconnect(String);
