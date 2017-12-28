#pragma once

#include "hornet.h"


class Worker
{
public:
    Worker(int master_fd);
    ~Worker();

    void operator ()();
    void Stop();

private:
    bool HandleServerMsg();
    bool HandleRequest(const Event& event);

    bool run_;
    int master_fd_;
    EventEngine event_;
};