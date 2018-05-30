#pragma once

#include "hornet.h"
#include "event.h"
#include "disk.h"
#include "accept_handler.h"
#include "client_handler.h"


class Worker: public EventEngine
{
public:
    Worker(int id, Disk *disk, map<string, string> &conf);

    bool Init();
    int GetSendMsgFd();

private:
    int msg_fd_[2]; // socket pair
    int id_;
    Disk *disk_;
    map<string, string> conf_;
};
