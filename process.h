#pragma once

#include "hornet.h"


class Process
{
public:
    Process(EventEngine *engine):engine_(engine){};
    virtual ~Process() {delete engine_;};

    void Forever() {engine_->Forever();};

private:
    EventEngine *engine_;
};


class Worker: public Process
{
public:
    Worker(Disk *disk, EventEngine *engine); // TODO: and socket pair
    ~Worker();

    bool SendHandler(Handler *handler);
    Handler* RecvHandler();

private:
    int socks[2]; // socket pair
};


class Master: public Process
{
public:
    Master(Disk *disk, EventEngine *engine); // TODO: and accept handler, and socket pair
    ~Master(){delete disk_; for (auto w: workers_){delete w;}};

private:
    Disk *disk_;
    vector<Worker*> workers_;
};
