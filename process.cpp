#include "process.h"


/*
 * worker
 */
Worker::Worker(int id):EventEngine()
{
    id_ = id;
    msg_fd_[0] = msg_fd_[1] = -1;
}


bool Worker::Init()
{
    if (socketpair(AF_UNIX, SOCK_STREAM|SOCK_NONBLOCK, 0, msg_fd_) < 0) {
        return false;
    }
    return true;
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
}


bool Master::Init()
{
    // DiskHandler *disk = new DiskHandler();
    // disk->fd = DISK_FD;
    // AddHandler(disk);
    // AddTimer(disk->fd, 10);

    AcceptHandler *accept = new AcceptHandler("0.0.0.0", 8080);

    if (accept == nullptr || ! accept->Init(this)) {
        return false;
    }

    return true;
}


void Master::Stop()
{
    for (size_t i = 0; i < workers_.size(); i++) {
        workers_[i]->Stop();
    }

    EventEngine::Stop();
}


void Master::AddWorker(Worker* worker)
{
    // workers_.push_back(unique_ptr<Worker>(worker));

    // MsgHandler *h = new MsgHandler();
    // h->fd = worker->GetSendMsgFd();

    // AddHandler(h);
    // AddEpollEvent(h->fd, EPOLLIN);
}
