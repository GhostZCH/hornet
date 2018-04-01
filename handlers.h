#pragma once

#include "hornet.h"


typedef struct sockaddr_in Address;
socklen_t ADDR_SIZE = sizeof(Address);


class AcceptHandler:public Handler
{
public:
    AcceptHandler();
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
class ＭsgHandler:public Handler
{
public:
    ＭsgHandler();
    ~ＭsgHandler();

   void Handle(const Event& ev, const EventEngine& engine);
};
