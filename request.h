#pragma once

#include "hornet.h"


// only allow GET PUT or DEL so the length of request line is fixed
const short METHOD_GET = 'G'; 
const short METHOD_PUT = 'P';
const short METHOD_DEL = 'D';


enum class HttpPhase
{
    RECV,
    SEND,
};


struct Request
{
    int fd;
    bool keep_alive;

    short method;
    short http_code;

    size_t length;
    size_t process_len;
    size_t content_len;

    Key key;
    Key dir;
    time_t addtime;

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
