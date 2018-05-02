#include "event.h"

Handler::~Handler()
{
    if (fd > 0) {
        close(fd);
    }
}


EventEngine::EventEngine(int connection_limit)
{
    run_ = false;
    con_limit_ = connection_limit;

    if ((epoll_ = epoll_create(con_limit_))< 0) {
        return;
    }

    run_ = true;
}


bool EventEngine::Forever()
{
    while (run_) {
        now_ = (uint32_t)time(nullptr);
        if (!HandleEpollEvent() || !HandleTimerEvent()) {
            return false;
        }
    }

    return true;
}


void EventEngine::Stop() {
    run_ = false;
}


bool EventEngine::AddHandler(Handler *handler)
{
    if (handlers_.size() >= (size_t)con_limit_) {
        return false;
    }

    if (handlers_.find(handler->fd) != handlers_.end()) {
        return false;
    }

    handlers_[handler->fd] = unique_ptr<Handler>(handler);
    return true;
}


bool EventEngine::DelHandler(int fd)
{
    if (handlers_.find(fd) == handlers_.end()) {
        return false;
    }

    handlers_.erase(fd);

    return true;
}


bool EventEngine::AddEpollEvent(int fd, int flag)
{
    struct epoll_event event; 
    event.data.fd = fd;
    event.events = flag;
    return epoll_ctl(epoll_, EPOLL_CTL_ADD, fd, &event) == 0;
}


bool EventEngine::DelEpollEvent(int fd)
{
    return epoll_ctl(epoll_, EPOLL_CTL_DEL, fd, NULL) == 0;
}


bool EventEngine::AddTimer(int fd, uint32_t timeout, int id)
{
    int64_t sub_id = fd;

    timers_[timeout + now_].insert(sub_id << 32 | id);
    return true;
}


bool EventEngine::DelTimer(int fd, uint32_t timeout, int id)
{
    int64_t sub_id = fd;
    timers_[timeout + now_].erase(sub_id << 32 | id);
    return true;
}


bool EventEngine::HandleEpollEvent()
{
    struct epoll_event epoll_events[EPOLL_WAIT_EVENTS];

    int n = epoll_wait(epoll_, epoll_events, EPOLL_WAIT_EVENTS, 200);

    for (int i = 0; i < n; i++) {
        auto iter = handlers_.find(epoll_events[i].data.fd);

        if (iter == handlers_.end()) {
            return false;
        }

        Event event = {0};
        event.read = (epoll_events[i].events & EPOLLIN) != 0;
        event.write = (epoll_events[i].events & EPOLLOUT) != 0;
        event.error = (epoll_events[i].events & (EPOLLHUP|EPOLLERR)) != 0;

        if (!iter->second->Handle(&event, this)) {
            if (!iter->second->Close(this)) {
                return false;
            }
        }
    }

    return true;
}


bool EventEngine::HandleTimerEvent()
{
    for (auto iter = timers_.begin(); iter != timers_.end(); ) {

        if (iter->first > now_) {
            break;
        }

        for (auto timer: iter->second) {
            int fd = timer >> 32 & 0xFFFFFFFF;

            auto iter = handlers_.find(fd);
            if (iter != handlers_.end()) {
                continue;
            }

            Event event = {0};
            event.timer = true;
            event.data.i = timer & 0xFFFFFFFF;
            iter->second->Handle(&event, this);
        }

        iter = timers_.erase(iter);
    }

    return true;
}
