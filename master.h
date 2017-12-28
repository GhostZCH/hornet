#pragma once

#include "hornet.h"

typedef unordered_map<int, unique_ptr<Worker>> WrokerMap;

class Master
{
public:
    Master(const string &ip, int port, int worker_count);
    ~Master();

    void Forever();

private:
    bool InitWorker();

    bool HandleAccept(Event& event);
    bool HandleClient(Event& event);
    bool HandleWorker(Event& event); // send request to worker recv finished request from worker

    bool run_;
    int server_fd_;

    EventEngine event_;
    WrokerMap workers_; // random select
};
