#include "master.h"



Master::Master(const string &ip, int port, int worker_count)
{
    run_ = false;
    server_fd_ = -1;

    if (!event_.Init()) {
        return;
    }

    Address addr;
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    addr.sin_addr.s_addr = inet_addr(ip.c_str());

    server_fd_ = socket(AF_INET, SOCK_STREAM, 0);
    if (server_fd_ < 0) {
        return;
    }
    
    if (fcntl(server_fd_, F_SETFL, fcntl(server_fd_, F_GETFL) | O_NONBLOCK) < 0) {
        return;
    }

    if (bind(server_fd_, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
        return;
    }

    if (listen(server_fd_, 1024) < 0) {
        return;
    }

    if (!event_.AddEvent(server_fd_, EPOLLIN)) {
        return;
    }

    for (unsigned int i = 0; i < worker_count; i++) {
        int fds[2];
        if (socketpair(AF_UNIX, SOCK_STREAM, 0, fds) < 0) {
            return;
        }

        // TODO: noblock
        Worker *worker = new Worker(fds[1]);
        workers_[fds[0]] = unique_ptr<Worker>(worker);
        event_.AddEvent(fds[0], EPOLLIN);
        thread t(*worker);
    }

    run_ = true;
}


Master::~Master() 
{
    if (server_fd_ > 0) {
        close(server_fd_);
    }
}


bool Master::HandleAccept(Event& event)
{
    Request r;

    while (true) {
        r.fd = accept4(server_fd_, (struct sockaddr*)&r.client_addr, &ADDR_SIZE, SOCK_NONBLOCK);

        if (r.fd > 0) {
            event_.AddEvent(r.fd, EPOLLET|EPOLLIN|EPOLLOUT);
            g_requests_map[r.fd] = r;
            continue;
        }

        if (r.fd < 0 && errno == EAGAIN) {
            return true;
        }

        return false;
    }

    return true;
}


bool Master::HandleClient(Event& event)
{
    Request *r = &g_requests_map[event.data.fd];
    
    if (event.events|EPOLLIN && r->phase == HttpPhase::RECV) {
        if (!RequestHandler::Read(r)) {
            // send to worker
            return false;
        }
    }

    return true;
}


void Master::Forever()
{
    Event events[EPOLL_WAIT_EVENTS];
    
    while (run_) {
        int n = event_.Wait(events, EPOLL_WAIT_EVENTS, 1000);
        if (n < 0) {
            return;
        }

        for (int i = 0; i < n; i++) {
            EventData d = events[i].data;
            if (events[i].data.fd == server_fd_) {
                if (!HandleAccept(events[i])) {
                    return;
                }
            } else if (workers_.find(fd) != workers_.end()) {
                if (!HandleWorker(events[i])){
                    return;
                }
            }else {
                if (!HandleClient(events[i])) {
                    g_requests_map.erase(events[i].data.fd);
                    close(events[i].data.fd);
                }
            }
        }
    }
}