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


bool EventEngine::Forever()
{
    while (run_) {
        update_time();
        unique_lock<mutex> run(run_lock);
        if (!HandleEpollEvent() || !HandleTimerEvent()) {
            return false;
        }
    }

    return true;
}


void EventEngine::Stop() {
    run_ = false;
}


bool EventEngine::AddHandler(shared_ptr<Handler>& h)
{
    if (handlers_.size() >= (size_t)con_limit_) {
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


bool EventEngine::AddTimer(int fd, time_t timeout, int id)
{
    int64_t sub_id = fd;

    timers_[timeout].insert(sub_id << 32 | id);
    return true;
}


bool EventEngine::DelTimer(int fd, time_t timeout, int id)
{
    auto iter = timers_.find(timeout);
    if (iter == timers_.end()) {
        return true;
    }

    int64_t tid = fd;
    tid = tid << 32 | id;
    if (iter->second.find(tid) == iter->second.end()) {
        return true;
    }

    timers_[timeout].erase(tid);
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
    list<int64_t> expired;

    time_t now = g_now;
    for (auto iter = timers_.begin(); iter->first < now && iter != timers_.end(); iter++) {
        for (auto h: iter->second) {
            expired.push_back(h);
        }
    }

    for (auto h: expired) {
        int fd = h >> 32 & 0xFFFFFFFF;

        auto iter = handlers_.find(fd);
        if (iter == handlers_.end()) {
            continue;
        }

        Event event = {0};
        event.timer = true;
        event.data.i = h & 0xFFFFFFFF;
        iter->second->Handle(&event, this);
    }

    return true;
}
