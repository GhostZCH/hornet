#pragma once

#include "hornet.h"


class Worker: public EventEngine
{
public:
    Worker(int id);
    int GetSendMsgFd();

private:
    int msg_fd_[2]; // socket pair
    int id_;
};


class Master: public EventEngine
{
public:
    Master(); // TODO: and accept handler, and socket pair
    void Stop();

    void AddWorker(Worker* worker);

private:
    unique_ptr<Disk> disk_;
    vector<unique_ptr<Worker>> workers_;
};
