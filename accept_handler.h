#pragma once

#include "hornet.h"
#include "event.h"
#include "disk.h"


class AcceptHandler:public Handler
{
public:
    AcceptHandler(const string& ip, short port);

    void Init(EventEngine* engine);
    void Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    string ip_;
    short port_;
    mutex accept_lock_;
};
