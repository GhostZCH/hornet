#pragma once

#include "event.h"


EventEngine::EventEngine(int connection_limit)
{
    run_ = false;
    con_limit_ = connection_limit;

    if ((epoll_ = epoll_create(con_limit_))< 0) {
        return;
    }

    run_ = true;
}


void EventEngine::Forever()
{
    while (run_) {
        now_ = (uint32_t)time(nullptr);

        HandleEpollEvent();
        HandleTimerEvent();
    }
}


bool EventEngine::AddHandler(Handler *h)
{
    if (nullptr == h || h->fd <= 0) {
        return false;
    }

    if (handlers_.size() >= con_limit_) {
        return false;
    }

    if (handlers_.find(h->fd) != handlers_.end()) {
        return false;
    }

    handlers_[h->fd] = unique_ptr<Handler>(h);
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
    EpollEvent event; 
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
    timers_[timeout + now_].insert(fd << 32 + id);
    return true;
}


bool EventEngine::DelTimer(int fd, uint32_t timeout, int id)
{
    timers_[timeout + now_].erase(fd << 32 + id);
    return true;
}


void EventEngine::HandleEpollEvent()
{
    struct epoll_event events[EPOLL_WAIT_EVENTS];

    int n = epoll_wait(epoll_, events, EPOLL_WAIT_EVENTS, 200);

    for (int i = 0; i < n; i++) {
        auto handler = handlers_.find(events[i].data.fd);

        if (handler !== handlers_.end()) {
            continue;
        }

        Event event;
        event.read = (events[i].events & EPOLLIN) != 0;
        event.write = (events[i].events & EPOLLOUT) != 0;
        event.error = (events[i].events & (EPOLLHUP|EPOLLERR)) != 0;
        event.timer = false;
        event.arg = 0

        handler->second.get()->Handle(event, *this);
    }
}


void EventEngine::HandleTimerEvent()
{
    auto iter = timers_.begin();

    for (; iter != timer_.end(); iter++) {
        if (iter.first > now_) {
            break;
        }

        for (auto timer: iter->second) {
            int fd = timer >> 32 & 0xFFFFFFFF;

            auto handler = handlers_.find(fd);
            if (handler !== handlers_.end()) {
                continue;
            } 

            Event event;

            ev.read = false;
            ev.write = false;
            ev.error = false;
            ev.timer = true;
            ev.arg = timer & 0xFFFFFFFF;

            handler->second.get()->Handle(event, *this);
        }
    }
}
