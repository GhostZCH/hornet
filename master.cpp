#include "master.h"
#include "tool.h"

void start_worker(Worker *w){
    w->Forever();
}

bool Master::Start()
{
    auto *d = new Disk(
        g_config["disk.path"],
        stoll(g_config["disk.block.count"]),
        stoll(g_config["disk.block.size"])
    );

    if (!d->Init()) {
        return false;
    }

    disk_ = unique_ptr<Disk>(d);

    short port = stoi(g_config["master.port"]);
    auto accepter = shared_ptr<Handler>(new AcceptHandler(g_config["master.ip"], port, d));
    if (accepter->fd < 0) {
        logger(LOG_ERROR, "listen " << g_config["master.ip"] << ":" << port << "failed");
        return false;
    }

    // auto *svr = new ServerHandler();
    // disk->fd = -2;
    // AddHandler(disk);
    // AddTimer(disk->fd, 10);

    int wc = stoi(g_config["worker.count"]);
    for (int i = 0; i < wc; i++) {
        Worker* w = new Worker(i, d);

        threads_.push_back((thread(start_worker, w)));
        workers_.push_back(unique_ptr<Worker>(w));
        if (!w->AddHandler(accepter) || !accepter->Init(w)) {
            return false;
        }
    }

    bool re = Forever();
    for (auto &t: threads_) {
        t.join();
    }
    return re;
}


void Master::Stop()
{
    EventEngine::Stop();
    for (auto &w: workers_) {
        w->Stop();
    }
}
