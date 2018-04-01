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

    handlers_[h->fd] = h;
    return true;
}


bool EventEngine::DelHandler(int fd)
{
    if (handlers_.find(fd) == handlers_.end()) {
        return false;
    }

    // need to release handler manually
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
        auto iter = handlers_.find(events[i].data.fd);

        if (iter != handlers_.end()) {
            continue;
        }

        Event event = {0};
        event.read = (events[i].events & EPOLLIN) != 0;
        event.write = (events[i].events & EPOLLOUT) != 0;
        event.error = (events[i].events & (EPOLLHUP|EPOLLERR)) != 0;

        iter->second->Handle(event, *this);
    }
}


void EventEngine::HandleTimerEvent()
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
            iter->second->Handle(event, *this);
        }

        iter = timers_.erase(iter);
    }
}
