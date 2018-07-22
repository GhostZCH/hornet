#include "accept_handler.h"
#include "client_handler.h"


AcceptHandler::AcceptHandler(const string& ip, short port)
    :Handler()
{
    ip_ = ip;
    port_ = port;

    struct sockaddr_in addr;
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port_);
    addr.sin_addr.s_addr = inet_addr(ip_.c_str());

    fd = socket(AF_INET, SOCK_STREAM|SOCK_NONBLOCK, 0);
    if (fd < 0) {
        return;
    }

    if (bind(fd, (struct sockaddr *)&addr, ADDR_SIZE) < 0 || listen(fd, 4096) < 0) {
        close(fd);
        fd = -1;
    }
}


bool AcceptHandler::Init(EventEngine *engine)
{
    return engine->AddTimer(fd, 0, 0);
}


bool AcceptHandler::Close(EventEngine *engine)
{
    return engine->DelEpollEvent(fd);
}


bool AcceptHandler::Handle(Event* ev, EventEngine* engine)
{
    if (ev->error) {
        return false;
    }

    unique_lock<mutex> ulock(accept_lock_, try_to_lock);
    if (!ulock) {
        return true;
    }

    LOG(LDEBUG, "worker" << engine << "getlock");

    int cfd;
    Address addr;
    socklen_t addr_size = sizeof(addr);

    while ((cfd = accept4(fd, &addr, &addr_size, SOCK_NONBLOCK)) > 0) {
        ClientHandler* client = new ClientHandler();
        client->fd = cfd;

        if (!client->Init(engine)) {
            LOG(LERROR, "client " << cfd << "init failed");
            return false;
        }
    }

    return errno == EAGAIN;
}
