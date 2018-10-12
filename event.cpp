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
        update_time();
        HandleEpollEvent();
        HandleTimerEvent();
    }
}


void EventEngine::Stop() {
    run_ = false;
}


void EventEngine::AddHandler(shared_ptr<Handler>& h)
{
    if (handlers_.size() >= (size_t)con_limit_) {
        throw ReqError("too many connections", __FILE__, __LINE__);
    }

    if (handlers_.find(h->fd) != handlers_.end()) {
        throw SvrError("fd exist fd = " + to_string(h->fd), __FILE__, __LINE__);
    }

    handlers_[h->fd] = h;
}


void EventEngine::DelHandler(int fd)
{
    if (handlers_.find(fd) == handlers_.end()) {
        throw SvrError("fd not exist fd = " + to_string(fd), __FILE__, __LINE__);
    }

    handlers_.erase(fd);
}


void EventEngine::AddEpollEvent(int fd, int flag)
{
    struct epoll_event event; 
    event.data.fd = fd;
    event.events = flag;
    if (epoll_ctl(epoll_, EPOLL_CTL_ADD, fd, &event) != 0) {
        throw ReqError("add to epoll fail fd = " + to_string(fd), __FILE__, __LINE__);
    }
}


void EventEngine::DelEpollEvent(int fd)
{
    if (epoll_ctl(epoll_, EPOLL_CTL_DEL, fd, NULL) != 0) {
        throw ReqError("delete from epoll fail fd = " + to_string(fd), __FILE__, __LINE__);
    }
}


void EventEngine::AddTimer(int fd, time_t timeout, int id)
{
    int64_t sub_id = fd;
    timers_[timeout].insert(sub_id << 32 | id);
}


void EventEngine::DelTimer(int fd, time_t timeout, int id)
{
    auto iter = timers_.find(timeout);
    if (iter == timers_.end()) {
        return;
    }

    int64_t tid = fd;
    timers_[timeout].erase(tid << 32 | id);
}


void EventEngine::HandleEpollEvent()
{
    struct epoll_event epoll_events[EPOLL_WAIT_EVENTS];

    int n = epoll_wait(epoll_, epoll_events, EPOLL_WAIT_EVENTS, 200);

    for (int i = 0; i < n; i++) {
        auto iter = handlers_.find(epoll_events[i].data.fd);
        if (iter == handlers_.end()) {
            throw SvrError("handler not exist fd = " + to_string(epoll_events[i].data.fd), __FILE__, __LINE__);
        }

        Event event = {0};
        event.read = (epoll_events[i].events & EPOLLIN) != 0;
        event.write = (epoll_events[i].events & EPOLLOUT) != 0;
        event.error = (epoll_events[i].events & (EPOLLHUP|EPOLLERR)) != 0;

        try {
            iter->second->Handle(&event, this);
        } catch (ReqError& err) {
            iter->second->Close(this);
        }
    }
}


void EventEngine::HandleTimerEvent()
{
    vector<int64_t> expired;

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
            throw SvrError("handler not exist fd = " + to_string(fd), __FILE__, __LINE__);
        }

        Event event = {0};
        event.timer = true;
        event.data.i = h & 0xFFFFFFFF;

        try {
            iter->second->Handle(&event, this);
        } catch (ReqError& err) {
            iter->second->Close(this);
        }
    }
}
