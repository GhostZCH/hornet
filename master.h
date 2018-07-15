#pragma once

#include "hornet.h"
#include "event.h"
#include "disk.h"
#include "accept_handler.h"
#include "client_handler.h"
#include "worker.h"


class Master: public EventEngine
{
public:
    bool Start();
    void Stop();

private:
    unique_ptr<Disk> disk_;
    vector<thread> threads_;
    vector<unique_ptr<Worker>> workers_;
};
 