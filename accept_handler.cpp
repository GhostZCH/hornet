#include "accept_handler.h"
#include "client_handler.h"


AcceptHandler::AcceptHandler(const string& ip, short port, Disk* disk)
{
    ip_ = ip;
    port_ = port;
    disk_ = disk;
}


bool AcceptHandler::Init(EventEngine *engine)
{
    struct sockaddr_in addr;

    addr.sin_family = AF_INET;
    addr.sin_port = htons(port_);
    addr.sin_addr.s_addr = inet_addr(ip_.c_str());

    fd = socket(AF_INET, SOCK_STREAM|SOCK_NONBLOCK, 0);

    if (fd < 0) {
        return false;
    }

    if (bind(fd, (struct sockaddr *)&addr, ADDR_SIZE) <0) {
        return false;
    }

    if (listen(fd, 4096) < 0) {
        return false;
    }

    if (!engine->AddHandler(this)) {
        return false;
    }

    if (!engine->AddEpollEvent(fd, EPOLLIN|EPOLLERR|EPOLLHUP)) {
        return false;
    }

    return true;
}


bool AcceptHandler::Close(EventEngine *engine)
{
    close(fd);
    return engine->DelEpollEvent(fd);
}


bool AcceptHandler::Handle(Event* ev, EventEngine* engine)
{
    if (ev->error || !ev->read) {
        return false;
    }

    while (true) {
        Address client_addr;
        socklen_t addr_size = sizeof(client_addr);

        int client_sock = accept4(fd, &client_addr, &addr_size, SOCK_NONBLOCK);
        if (client_sock < 0) {
            break;
        }

        ClientHandler *client = new ClientHandler(disk_);
        client->fd = client_sock;

        if (!client->Init(engine)) {
            return false;
        }
    }

    return true;
}
