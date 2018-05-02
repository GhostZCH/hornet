#pragma once

#include "hornet.h"
#include "item.h"
#include "event.h"
#include "pool.h"
#include "buffer.h"

enum HttpMethod {
    HTTP_GET,
    HTTP_POST,
    HTTP_PUT,
    HTTP_DELETE
};




class ClientHandler:public Handler
{
public:
    bool Init(EventEngine* engine);
    bool Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    bool reading_{true};

    HttpMethod method_{HTTP_GET};

    Pool* pool_;

    Item *item_{nullptr};
};
