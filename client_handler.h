#pragma once

#include "hornet.h"
#include "item.h"
#include "event.h"
#include "disk.h"


class ClientHandler:public Handler
{
public:
    ClientHandler(Disk* disk, size_t buf_cap);

    bool Init(EventEngine* engine);
    bool Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    bool handleRead();
    bool handleWrite();

    bool processReqLine(char *header_end);
    bool setRspHeader();

    bool reading_{true};

    int method_{0};
    map<string, ssize_t> args_;
    uint16_t state_;

    bool send_disk_{false};
    unique_ptr<char []> buf_;
    size_t process_len_{0};
    size_t buf_size_{0};
    size_t buf_capacity_{0};

    size_t content_len{0};

    Disk* disk_;
    Item *item_{nullptr};
};
