#include "worker.h"


Worker::Worker(int master_fd)
{
    run_ = false;
    master_fd_ = master_fd;    
    if (!event_.Init()) {
        return;
    }

    if (!event_.AddEvent(master_fd_, EPOLLIN)) {
        return;
    }

    run_ = true;
}


Worker::~Worker()
{
    close(master_fd_);
}


void Worker::Stop()
{
    run_ = false;
}


void Worker::operator()()
{
    Event events[EPOLL_WAIT_EVENTS];

    while (run_) {
        int n = event_.Wait(events, EPOLL_WAIT_EVENTS, 1000);
        if (n < 0) {
            return;
        }

        for (int i = 0; i < n; i++) {
            if (events[i].data.fd == master_fd_) {
                HandleServerMsg();
            } else {
                HandleRequest(events[i]);
            }
        }
    }
}


bool Worker::HandleRequest(const Event& event)
{
    Request *r = &g_requests_map[event.data.fd];
    
    if (event.events|EPOLLIN && r->phase == HttpPhase::RECV) {
        if (!RequestHandler::Read(r)) {
            return false;
        }
    }

    if (event.events|EPOLLOUT && r->phase == HttpPhase::SEND) {
        if (!RequestHandler::Write(r)) {
            return false;
        }
    }
}


bool Worker::HandleServerMsg()
{
    const int FD_LIMIT = 1024;
    int fds[FD_LIMIT];

    while(true) {
        //TODO: recv from master
    }

    return true;
}
