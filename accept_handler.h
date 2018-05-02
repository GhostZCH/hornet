#pragma once

#include "hornet.h"
#include "event.h"

class AcceptHandler:public Handler
{
public:
    AcceptHandler(const string& ip, short port);

    bool Init(EventEngine* engine);
    bool Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    string ip_;
    short port_;
};
