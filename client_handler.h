#pragma once

#include "hornet.h"
#include "item.h"
#include "event.h"
#include "disk.h"


class ClientHandler:public Handler
{
public:
    ClientHandler(Disk* disk);

    bool Init(EventEngine* engine);
    bool Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    bool Read();
    bool Write();

    bool ProcessInput();
    bool GetSpecialHeader();

    bool reading_{true};

    int method_{0};
    uint16_t state_;

    unique_ptr<char []> buf_;
    size_t process_len_{0};
    size_t buf_size_{0};
    size_t buf_capacity_{HEADER_SIZE};

    size_t content_len{0};

    Disk* disk_;
    Item *item_{nullptr};
};
