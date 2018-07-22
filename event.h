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

    virtual bool Init(EventEngine* engine) = 0;
    virtual bool Close(EventEngine* engine) = 0;

    virtual bool Handle(Event* ev, EventEngine* engine) = 0;

    int fd{-1};
};


class EventEngine
{
public:
    EventEngine(int connection_limit=10240);

    bool Forever();
    virtual void Stop();

    bool AddHandler(shared_ptr<Handler>& h);
    bool DelHandler(int fd);

    bool AddEpollEvent(int fd, int flag=EPOLLIN|EPOLLOUT|EPOLLET|EPOLLHUP|EPOLLERR);
    bool DelEpollEvent(int fd);

    bool AddTimer(int fd, uint32_t timeout, int id=0);
    bool DelTimer(int fd, uint32_t timeout, int id=0);

    uint32_t Now();

    mutex run_lock;
    map<string, void*> context;

protected:
    bool HandleEpollEvent();
    bool HandleTimerEvent();

    bool run_;

    // for net io events
    int epoll_;
    int con_limit_;

    // for timer events
    uint32_t resolution_;
    map<uint32_t, unordered_set<int64_t>> timers_;

    // handler {fd: handler}
    unordered_map<int, shared_ptr<Handler>> handlers_;
};
