#include "master.h"
#include "tool.h"


bool Master::Start()
{
    auto *d = new Disk(
        g_config["disk.path"],
        stoll(g_config["disk.block.count"]),
        stoll(g_config["disk.block.size"])
    );

    if (!d->Init()) {
        LOG(LERROR, "disk init failed");
        return false;
    }

    disk_ = unique_ptr<Disk>(d);

    short port = stoi(g_config["master.port"]);
    auto accepter = shared_ptr<Handler>(new AcceptHandler(g_config["master.ip"], port));
    if (accepter->fd < 0) {
        LOG(LERROR, "listen " << g_config["master.ip"] << ":" << port << "failed");
        return false;
    }

    int access = open(g_config["access"].c_str(), O_APPEND|O_CREAT, S_IRUSR|S_IWUSR);
    if (access < 0) {
        LOG(LERROR, "open access log file" << g_config["access"] << "failed");
        return false;
    }

    int wc = stoi(g_config["worker.count"]);
    for (int i = 0; i < wc; i++) {
        AccessLog* l = new AccessLog(access);
        EventEngine* w = new EventEngine();
        w->context["disk"] = d;
        w->context["access"] = l;

        if (!w->AddHandler(accepter) || !accepter->Init(w)) {
            return false;
        }

        loggers_.push_back(unique_ptr<AccessLog>(l));
        workers_.push_back(unique_ptr<EventEngine>(w));

        threads_.push_back(thread([w]()->void{w->Forever();}));
    }

    // auto *svr = new ServerHandler();
    // disk->fd = -2;
    // AddHandler(disk);
    // AddTimer(disk->fd, 10);

    bool re = Forever();
    Stop();
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
