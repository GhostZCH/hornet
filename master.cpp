#include "master.h"


Master::Master(map<string, string> &conf):EventEngine()
{
    conf_ = conf;
}


bool Master::Init()
{
    now_ = time(NULL);

    disk_ = unique_ptr<Disk>(new Disk(
        conf_["disk.path"],
        (uint32_t)stoll(conf_["disk.block.count"]),
        (size_t)stoll(conf_["disk.block.size"])
    ));

    if (!disk_->Init()) {
        return false;
    }

    disk_->UpdateTime(now_);

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
        disk_.get(),
        stoul(conf_["request.max_header_size"])
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
