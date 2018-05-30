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
    Master(map<string, string> &conf);

    bool Init();
    void Stop();

private:
    map<string, string> conf_;
    unique_ptr<Disk> disk_;
    vector<unique_ptr<Worker>> workers_;
};
