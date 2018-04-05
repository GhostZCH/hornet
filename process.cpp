#include "process.h"


/*
 * worker
 */
Worker::Worker(int id):EventEngine()
{
    id_ = id;
    if (socketpair(AF_UNIX, SOCK_STREAM|SOCK_NONBLOCK, 0, msg_fd_) < 0) {
        // log
        msg_fd_[0] = msg_fd_[1] = -1;
        Stop();
    }
}


int Worker::GetSendMsgFd()
{
    return msg_fd_[0];
}


/*
 * master
 */

Master::Master():EventEngine()
{
    DiskHandler *disk = new DiskHandler();
    disk->fd = DISK_FD;
    AddHandler(disk);
    AddTimer(disk->fd, 10);

    AcceptHandler *accept = new AcceptHandler();
    AddHandler(accept);
    AddEpollEvent(accept->fd, EPOLLIN|EPOLLERR|EPOLLHUP);
}


void Master::Stop()
{
    for (int i = 0; i < workers_.size(); i++) {
        (*workers_[i]).Stop();
    }

    EventEngine::Stop();
}


void Master::AddWorker(Worker* worker)
{
    workers_.push_back(unique_ptr<Worker>(worker));

    MsgHandler *h = new MsgHandler();
    h->fd = worker->GetSendMsgFd();

    AddHandler(h);
    AddEpollEvent(h->fd, EPOLLIN);
}
