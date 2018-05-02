#pragma once

#include "hornet.h"
#include "event.h"
#include "disk.h"
#include "accept_handler.h"
#include "client_handler.h"

class Worker: public EventEngine
{
public:
    Worker(int id);

    bool Init();
    int GetSendMsgFd();

private:
    int msg_fd_[2]; // socket pair
    int id_;
};


class Master: public EventEngine
{
public:
    Master();

    bool Init();
    void Stop();

    void AddWorker(Worker* worker);

private:
    unique_ptr<Disk> disk_;
    vector<unique_ptr<Worker>> workers_;
};
