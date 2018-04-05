#pragma once

#include "hornet.h"


AcceptHandler::AcceptHandler(const string& ip, short port)
{
    struct sockaddr_in srv;
    socklen_t len = sizeof(srv);
    srv.sin_family = AF_INET;
    srv.sin_port = htons(port);
    srv.sin_addr.s_addr = inet_addr(ip.c_str());

    fd = socket(AF_INET, SOCK_STREAM|SOCK_NONBLOCK, 0);
    if (fd < 0) {
        return;
    }

    bind()
}


AcceptHandler::~AcceptHandler()
{
    close(fd);
}


void AcceptHandler::Handle(const Event& ev, const EventEngine& engine)
{

}



class ClientHandler:public Handler
{
public:
    ClientHandler();
    ~ClientHandler();

   void Handle(const Event& ev, const EventEngine& engine);
};


// for inner msg across master and workers
class MsgHandler:public Handler
{
public:
    MsgHandler();
    ~MsgHandler();

   void Handle(const Event& ev, const EventEngine& engine);
};



// disk periodic task
class DiskHandler:public Handler
{
public:
    DiskHandler();
    ~DiskHandler();

   void Handle(const Event& ev, const EventEngine& engine);
};

