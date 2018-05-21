#pragma once

#include "hornet.h"
#include "event.h"
#include "disk.h"

class AcceptHandler:public Handler
{
public:
    AcceptHandler(const string& ip, short port, Disk* disk);

    bool Init(EventEngine* engine);
    bool Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    string ip_;
    short port_;
    Disk* disk_;
};
