#pragma once

#include "hornet.h"
#include "item.h"
#include "disk.h"
#include "event.h"
#include "access_log.h"
#include "request.h"


class ClientHandler:public Handler
{
public:
    ClientHandler();

    bool Init(EventEngine* engine);
    bool Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    time_t timeout_;
    unique_ptr<Request> req_;
};
