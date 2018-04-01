#pragma once

#include "hornet.h"


class EventEngine;


struct Event
{
    bool read;
    bool write;
    bool error;
    bool timer;
    union {
        int i;
        float f;
        void* p;
    } data;
};


class Handler
{
public:
    Handler() {};
    virtual ~Handler(){};
    virtual void Handle(const Event& ev, const EventEngine& engine){};

    int fd;
};


class EventEngine
{
public:
    EventEngine(int connection_limit=10240);
    ~EventEngine(){};

    void Forever();

    bool AddHandler(Handler *h);
    bool DelHandler(int fd);

    bool AddEpollEvent(int fd, int flag=EPOLLIN|EPOLLOUT|EPOLLET|EPOLLHUP|EPOLLERR);
    bool DelEpollEvent(int fd);

    bool AddTimer(int fd, uint32_t timeout);
    bool DelTimer(int fd, uint32_t timeout);
    uint32_t Now();

private:
    void HandleEpollEvent();
    void HandleTimerEvent();

    bool run_;

    // for net io events
    int epoll_;
    int con_limit_;

    // for timer events
    uint32_t now_;
    uint32_t resolution_;
    map<uint32_t, unordered_set<int64_t>> timers_;

    // handler {fd: handler}
    unordered_map<int, unique_ptr<Handler>> handlers_;
};
