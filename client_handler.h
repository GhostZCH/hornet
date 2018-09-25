#pragma once

#include "hornet.h"
#include "item.h"
#include "disk.h"
#include "event.h"
#include "access_log.h"

enum Phase
{
    PH_READ_HEADER,
    PH_READ_BODY,
    PH_SEND_MEM,
    PH_SEND_DISK,
    PH_FINISH,
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
    time_t start;
    uint16_t state;
    uint32_t header_len;
    uint32_t write_len;
    uint32_t content_len;
    map<string, string> args;
    map<string, string> headers;
    string method_str;
    string uri_str;
    const char* error;
};


class ClientHandler:public Handler
{
public:
    ClientHandler();

    bool Init(EventEngine* engine);
    bool Close(EventEngine* engine);

    bool Handle(Event* ev, EventEngine* engine);

private:
    void readHeader(Event* ev, EventEngine* engine);
    void readBody(Event* ev, EventEngine* engine);
    void sendMem(Event* ev, EventEngine* engine);
    void sendDisk(Event* ev, EventEngine* engine);
    void finish(Event* ev, EventEngine* engine);
    void timeout(Event* ev, EventEngine* engine);

    void handleRequest();
    void getItem();
    void addItem();
    void delItem();

    void reset();

    Phase phase_{PH_READ_HEADER};

    Buffer recv_;
    Buffer send_;
    Request req_;
    time_t timeout_;

    Disk* disk_;
    AccessLog* logger_;
    shared_ptr<Item> item_;
    shared_ptr<Block> block_;
};
