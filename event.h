#pragma once

#include "hornet.h"

using namespace std;

enum EventType{
    EV_TYPE_LISTEN,
    EV_TYPE_CLIENT,
    EV_TYPE_INNER
};


struct Event{
    int fd;
    int type;
};


class EventEngine
{
public:
    EventEngine(int max_events, int timeout);
    ~EventEngine();

    bool Init();
    void Forever();
    bool AddEvent(Event, int flag);
    bool DeleteEvent(int fd);
    int Wait(Event* events, , );

private:
    int epoll_;
    int max_events_;
    int timeout_;
};