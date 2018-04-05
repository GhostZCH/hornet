#pragma once

#include "hornet.h"


typedef struct sockaddr_in Address;
socklen_t ADDR_SIZE = sizeof(Address);

const int DISK_FD = -1; // fake fd for disk handler


class AcceptHandler:public Handler
{
public:
    AcceptHandler(const string& ip, short port);
    ~AcceptHandler();

   void Handle(const Event& ev, const EventEngine& engine);
};


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

