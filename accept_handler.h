#pragma once

#include "hornet.h"
#include "event.h"
#include "disk.h"

class AcceptHandler:public Handler
{
public:
    AcceptHandler(const string& ip, short port, Disk* disk, size_t buf_cap);

    bool Init(EventEngine* engine);
    bool Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    string ip_;
    short port_;

    // for client
    Disk* disk_;
    size_t buf_cap_{0};
};
