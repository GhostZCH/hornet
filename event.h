#pragma once

#include "hornet.h"
#include "tool.h"

struct Event
{
    bool read;
    bool write;
    bool error;
    bool timer;

    union {
        int64_t i;
        double d;
        void* p;
    } data;
};


class EventEngine;


class Handler
{
public:
    virtual ~Handler(){if (fd > 0) {close(fd);}}

    virtual void Init(EventEngine* engine) = 0;
    virtual void Close(EventEngine* engine) = 0;

    virtual void Handle(Event* ev, EventEngine* engine) = 0;

    int fd{-1};
};


class EventEngine
{
public:
    EventEngine(int connection_limit=10240);

    void Forever();
    virtual void Stop();

    void AddHandler(shared_ptr<Handler>& h);
    void DelHandler(int fd);

    void AddEpollEvent(int fd, int flag=EPOLLIN|EPOLLOUT|EPOLLET|EPOLLHUP|EPOLLERR);
    void DelEpollEvent(int fd);

    void AddTimer(int fd, time_t timeout, int id=0);
    void DelTimer(int fd, time_t timeout, int id=0);

    map<string, void*> context;

protected:
    void HandleEpollEvent();
    void HandleTimerEvent();

    bool run_;

    // for net io events
    int epoll_;
    int con_limit_;

    // for timer events
    time_t resolution_;
    map<time_t, unordered_set<int64_t>> timers_;

    // handler {fd: handler}
    unordered_map<int, shared_ptr<Handler>> handlers_;
};
