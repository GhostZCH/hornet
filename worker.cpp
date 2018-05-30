#include "worker.h"


/*
 * worker
 */
Worker::Worker(int id, Disk *disk, map<string, string> &conf): EventEngine()
{
    id_ = id;
    msg_fd_[0] = msg_fd_[1] = -1;
    disk_ = disk;
    conf_ = conf;
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
