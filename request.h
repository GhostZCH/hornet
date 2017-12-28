#pragma once

#include "hornet.h"


enum class HttpPhase
{
    RECV,
    SEND,
};


struct Request
{
    int fd;
    bool keep_alive;

    char version; // '0'=http1.0, '1'=http/1.1
    short method;
    short http_code;
    time_t expired;

    size_t process_len;
    size_t length;
    size_t content_length;

    Key key;
    Key dir;

    HttpPhase phase;
    Address client_addr;

    char buffer[BUF_SIZE];
};


class RequestHandler
{
public:
    static bool Init(Request *requst);
    static bool ParseRequestHeader(Request *requst);
    static bool GenrateResponseHeader(Request *requst); 
    static bool Read(Request *requst);
    static bool Write(Request *requst);
    static bool FinishRequest(Request *requst); 
};
