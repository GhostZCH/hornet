#pragma once

#include "hornet.h"
#include "event.h"
#include "disk.h"
#include "accept_handler.h"
#include "client_handler.h"


class Worker: public EventEngine
{
public:
    Worker(int id, Disk *disk);

private:
    int id_;
};
