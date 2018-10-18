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
    void Start();
    void Stop();
    void Reopen();

private:
    unique_ptr<Disk> disk_;
    vector<thread> threads_;
    vector<unique_ptr<EventEngine>> workers_;
    vector<unique_ptr<AccessLog>> access_log_;
};


void Hornet::Start()
{
    disk_ = unique_ptr<Disk>(new Disk(
        get_conf("disk.path"),
        stoll(get_conf("disk.block.count")),
        stoll(get_conf("disk.block.size"))
    ));

    disk_->Init();

    short port = stoi(get_conf("master.port"));
    auto accepter = shared_ptr<Handler>(new AcceptHandler(get_conf("master.ip"), port));
    if (accepter->fd < 0) {
        string err = "can not listen " + get_conf("master.ip") + ":" + to_string(port);
        throw SvrError(err, __FILE__, __LINE__);
    }

    int wc = stoi(get_conf("worker.count"));
    for (int i = 0; i < wc; i++) {
        access_log_.push_back(unique_ptr<AccessLog>(new AccessLog(get_conf("log.access"))));
        auto log = access_log_[access_log_.size() - 1].get();
        log->Init();

        workers_.push_back(unique_ptr<EventEngine>(new EventEngine()));
        auto worker = workers_[workers_.size() - 1].get();
        worker->context["disk"] = disk_.get();
        worker->context["access"] = log;
        worker->AddHandler(accepter);
        accepter->Init(worker);
        
        threads_.push_back(thread([worker]()->void{worker->Forever();}));
    }

    // auto *svr = new ServerHandler();
    // disk->fd = -2;
    // AddHandler(disk);
    // AddTimer(disk->fd, 10);

    LOG(LWARN, "start handle requests");
    Forever();

    for (auto &w: workers_) {
        w->Stop();
    }

    for (auto &t: threads_) {
        t.join();
    }

    LOG(LWARN, "process exit");
}


void Hornet::Stop()
{
    LOG(LWARN, "stop handle requests");
    EventEngine::Stop();
}


void Hornet::Reopen()
{
    LOG(LWARN, "reopen log files");
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
    unique_ptr<ofstream> errlog;

    try {
        update_time();
        set_logger("ERROR", &cerr);

        map<string, pair<string, string>> params;
        params["c"] = make_pair<string, string>("config file of hornet", "hornet.conf");
        get_param(argc, argv, params);
        load_conf(params["c"].second);

        errlog = unique_ptr<ofstream>(new ofstream(get_conf("log.error"), ios_base::app));
        if (!errlog->is_open() || !set_logger(get_conf("log.level"), errlog.get())) {
            throw SvrError("open errlog failed", __FILE__, __LINE__);
        }

        server = unique_ptr<Hornet>(new Hornet());

        if (signal(SIGTERM, signal_handler) == SIG_ERR 
            || signal(SIGINT, signal_handler) == SIG_ERR
            || signal(SIGUSR1, signal_handler) == SIG_ERR) {
            throw SvrError("add signal failed", __FILE__, __LINE__);
        }

        server->Start();
        return 0;
    } catch (SvrError & exc) {
        LOG(LERROR, exc);
    } catch (exception & exc) {
        LOG(LERROR, exc.what());
    } 

    return 1;
}
