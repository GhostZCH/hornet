#pragma once

#include "hornet.h"
#include "item.h"
#include "disk.h"
#include "event.h"


enum Phase
{
    PH_READ_HEADER,
    PH_READ_BODY,
    PH_SEND_MEM,
    PH_SEND_DISK
};


struct Buffer
{
    size_t size;
    size_t capcity;
    size_t offset;
    size_t processed;
    unique_ptr<char []> data;
};


struct Request
{
    int method;
    size_t id;
    size_t dir;
    uint16_t state;
    uint32_t header_len;
    uint32_t write_len;
    uint32_t content_len;
    map<string, string> args;
    map<string, string> headers;
};


class ClientHandler:public Handler
{
public:
    ClientHandler(Disk* disk);

    bool Init(EventEngine* engine);
    bool Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    bool readHeader(Event* ev, EventEngine* engine);
    bool readBody(Event* ev, EventEngine* engine);
    bool sendMem(Event* ev, EventEngine* engine);
    bool sendDisk(Event* ev, EventEngine* engine);

    bool handleRequest();
    bool getItem();
    bool addItem();
    bool delItem();

    void reset();

    Phase phase_{PH_READ_HEADER};

    Buffer recv_;
    Buffer send_;
    Request req_;

    Disk* disk_;
    shared_ptr<Item> item_;
    shared_ptr<Block> block_;
};
