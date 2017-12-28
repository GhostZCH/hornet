#include "event.h"

using namespace std;


EventData::EventData(const EpollData& data)
{
    uint32_t* u = (uint32_t*)&data.u64;
    fd = u[0];
    type = (FdType)u[1];
}


EventData::EventData(int fd, FdType type)
{
    fd = fd;
    type = type;
}


EpollData EventData::ToEpollData()
{
    EpollData data;
    data.u64 = fd << 32 + type;
    return data;
}


EventEngine::EventEngine()
{
    epoll_ = -1;
};


EventEngine::~EventEngine()
{
    if (epoll_ > 0) {
        close(epoll_);
    }
}


bool EventEngine::Init(){
    epoll_ = epoll_create(4096);
    return epoll_ > 0;
}


bool EventEngine::AddEvent(int fd, FdType type, int flag)
{
    Event ev;
    ev.data.u64 = fd;
    ev.events = flag;
    return epoll_ctl(epoll_, EPOLL_CTL_ADD, fd, &ev) >= 0;
}


bool EventEngine::DeleteEvent(int fd)
{
    return epoll_ctl(epoll_, EPOLL_CTL_DEL, fd, NULL) >= 0;
}


int EventEngine::Wait(Event* events, int max_events, int timeout)
{
    return epoll_wait(epoll_, events, max_events, timeout);
}
