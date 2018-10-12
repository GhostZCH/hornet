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

    void Init(EventEngine* engine);
    void Close(EventEngine* engine);

    void Handle(Event* ev, EventEngine* engine);

private:
    time_t timeout_;
    unique_ptr<Request> req_;
};
