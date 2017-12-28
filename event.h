#pragma once

#include "hornet.h"

using namespace std;

 
// same size of epoll_event.data
struct EventData{
    int fd;
    FdType type;

    EventData(const EpollData& event);
    EventData(int fd, FdType type);
    EpollData ToEpollData();
};


class EventEngine
{
public:
    EventEngine();
    ~EventEngine();

    bool Init();
    void Forever();
    bool AddEvent(int fd, FdType type, int flag);
    bool DeleteEvent(int fd, FdType type);
    int Wait(Event* events, int max_events, int timeout);
    
private:
    int epoll_;
};