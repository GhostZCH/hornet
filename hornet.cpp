#include "hornet.h"
#include "tool.h"
#include "event.h"
#include "disk.h"
#include "client_handler.h"
#include "accept_handler.h"

const char* VERSION_STR = "1.0.0";
const int VERSION = 10000;


class Hornet: public EventEngine
{
public:
    bool Start();
    void Stop();
    void Reopen();

private:
    unique_ptr<Disk> disk_;
    vector<thread> threads_;
    vector<unique_ptr<EventEngine>> workers_;
    vector<unique_ptr<AccessLog>> access_log_;
};


bool Hornet::Start()
{
    disk_ = unique_ptr<Disk>(new Disk(
        get_conf("disk.path"),
        stoll(get_conf("disk.block.count")),
        stoll(get_conf("disk.block.size"))
    ));

    if (!disk_->Init()) {
        LOG(LERROR, "disk init failed");
        return false;
    }

    short port = stoi(get_conf("master.port"));
    auto accepter = shared_ptr<Handler>(new AcceptHandler(get_conf("master.ip"), port));
    if (accepter->fd < 0) {
        LOG(LERROR, "listen " << get_conf("master.ip") << ":" << port << "failed");
        return false;
    }

    int wc = stoi(get_conf("worker.count"));
    for (int i = 0; i < wc; i++) {
        auto log = unique_ptr<AccessLog>(new AccessLog(get_conf("log.access")));
        if (!log->Init()) {
            return false;
        }
        access_log_.push_back(log);

        EventEngine* worker = new EventEngine();
        worker->context["disk"] = disk_.get();
        worker->context["access"] = log.get();

        if (!worker->AddHandler(accepter) || !accepter->Init(worker)) {
            return false;
        }

        workers_.push_back(unique_ptr<EventEngine>(worker));
        threads_.push_back(thread([worker]()->void{worker->Forever();}));
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


void Hornet::Stop()
{
    EventEngine::Stop();
    for (auto &w: workers_) {
        w->Stop();
    }
}


void Hornet::Reopen()
{
    for (auto &log: access_log_) {
        log->Reopen();
    }
}


unique_ptr<Hornet> server;


void signal_handler(int sig)
{
    LOG(LERROR, "signal_handler: " << sig);

    // reopen log file
    if (sig == SIGUSR1) {
        server->Reopen();
        return;
    }

    server->Stop();
}


int main(int argc, char* argv[])
{
    try {
        update_time();
        set_logger("ERROR", &cerr);

        map<string, pair<string, string>> params;
        params["c"] = make_pair<string, string>("config file of hornet", "hornet.conf");
        if (!get_param(argc, argv, params)) {
            return 1;
        }

        if (!load_conf(params["c"].second)) {
            LOG(LERROR, "load_conf failed");
            return 1;
        }

        ofstream errlog = ofstream(get_conf("log.error"), ios_base::app);
        if (!errlog.is_open() || !set_logger(get_conf("log.level"), &errlog)) {
            LOG(LERROR, "open errlog failed");
            return 1;
        }

        server = unique_ptr<Hornet>(new Hornet());

        if (signal(SIGTERM, signal_handler) == SIG_ERR 
            || signal(SIGINT, signal_handler) == SIG_ERR
            || signal(SIGUSR1, signal_handler) == SIG_ERR) {
            LOG(LERROR, "setup signal failed");
            return 1;
        }

        return server->Start() ? 0 : 1;

    } catch (const exception & exc) {

        LOG(LERROR, exc.what());
        return 1;
    }
}
