#pragma once

#include "hornet.h"
#include "item.h"
#include "disk.h"
#include "access_log.h"
#include "buffer.h"


const int STATUS_OK = 200;
const int STATUS_CREATED = 201;
const int STATUS_NOT_FOUND = 404;
const int STATUS_SERVER_ERROR = 500;


enum ReqPhase
{
    PH_READ_HEADER,
    PH_READ_BODY,
    PH_SEND_RSP,
    PH_SEND_CACHE,
    PH_FINISH
};


class Request
{
public:
    Request(int fd, Disk *d, AccessLog* log);
    ~Request();

    ReqPhase Phase(){return phase_;}
    void Error(const string& msg);

    bool ReadHeader();
    bool ReadBody();
    bool SendResponse();
    bool SendCache();
    bool Finish();


private:
    void parseReqLine(const char* &args, const char* &headers);
    void parseHeaders(const char* headers);
    void parseArgs(const char* args);
    void parseTags(uint16_t tags[]);
    void log();

    void getItem();
    void addItem();
    void delItem();

private:
    int fd_;
    Disk* disk_;
    AccessLog* log_;

    ReqPhase phase_;
    time_t start_;

    size_t id_;
    size_t dir_;

    uint16_t state_;
    uint32_t header_len_;

    unique_ptr<Buffer> recv_;
    unique_ptr<Buffer> send_;

    string uri_;
    string method_;
    string client_ext_;
    string server_ext_;

    string error_;

    map<string, string> args_;
    map<string, string> headers_;
};