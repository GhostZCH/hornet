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
    if (fd < 0 || bind(fd, (struct sockaddr *)&addr, ADDR_SIZE) < 0 || listen(fd, 4096) < 0) {
        if (fd > 0) {
            close(fd);
        }
        throw SvrError("AcceptHandler Handle failed: fd=" + to_string(fd), __FILE__, __LINE__);
    }
}


void AcceptHandler::Init(EventEngine *engine)
{
    engine->AddTimer(fd, 0, 0);
}


void AcceptHandler::Close(EventEngine *engine)
{
    engine->DelEpollEvent(fd);
}


bool AcceptHandler::Handle(Event* ev, EventEngine* engine)
{
    if (ev->error) {
        throw SvrError("AcceptHandler Handle failed", __FILE__, __LINE__);
    }

    unique_lock<mutex> ulock(accept_lock_, try_to_lock);
    if (!ulock) {
        return true;
    }

    int cfd;
    Address addr;
    socklen_t addr_size = sizeof(addr);

    while ((cfd = accept4(fd, &addr, &addr_size, SOCK_NONBLOCK)) > 0) {
        ClientHandler* client = new ClientHandler();
        client->fd = cfd;

        client->Init(engine);
    }

    if (errno != EAGAIN) {
        throw SvrError("AcceptHandler Handle failed", __FILE__, __LINE__);
    }

    return true;
}
