#include "event.h"

using namespace std;


EventEngine::EventEngine(int max_events, int timeout)
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


bool EventEngine::AddEvent(int fd, int type, int flag)
{
    Event ev;
    ev.data.u64 = type << 32 + fd;
    ev.events = flag;
    return epoll_ctl(epoll_, EPOLL_CTL_ADD, fd, &ev) >= 0;
}


bool EventEngine::DeleteEvent(int fd)
{
    return epoll_ctl(epoll_, EPOLL_CTL_DEL, fd, NULL) >= 0;
}


int EventEngine::Wait(EpEvent* events, int max_events)
{
    return epoll_wait(epoll_, events, max_events, timeout_);
}
