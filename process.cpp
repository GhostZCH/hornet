#include "process.h"


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


/*
 * master
 */

Master::Master(map<string, string> &conf):EventEngine()
{
    conf_ = conf;
}


bool Master::Init()
{
    disk_ = unique_ptr<Disk>(new Disk(
        conf_["disk.path"],
        (uint32_t)stoll(conf_["disk.block.count"]),
        0
    ));

    if (!disk_->Init()) {
        return false;
    }

    for (int id = 0; id < stoi(conf_["worker.count"]); id++) {
        auto w = new Worker(id, disk_.get(), conf_);
        if (!w->Init()) {
            return false;
        }
        workers_.push_back(unique_ptr<Worker>(w));
    }

    // DiskHandler *disk = new DiskHandler();
    // disk->fd = DISK_FD;
    // AddHandler(disk);
    // AddTimer(disk->fd, 10);

    AcceptHandler *accept = new AcceptHandler(
        conf_["master.ip"],
        stoi(conf_["master.port"]),
        disk_.get()
    );

    if (!accept->Init(this)) {
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
