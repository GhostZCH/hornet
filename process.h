#pragma once

#include "hornet.h"

class Process
{
public:
    Process();
    virtual ~Process();

    void Forever();

private:
    EventEngine *engine_;
};


class Master: public Process
{
public:
    Master();
    ~Master();

private:
};


class Worker: public Process
{

};
