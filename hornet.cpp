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

private:
    unique_ptr<Disk> disk_;
    vector<thread> threads_;
    vector<unique_ptr<EventEngine>> workers_;
    vector<unique_ptr<AccessLog>> loggers_;
};


bool Hornet::Start()
{
    auto *d = new Disk(
        get_conf("disk.path"),
        stoll(get_conf("disk.block.count")),
        stoll(get_conf("disk.block.size"))
    );

    if (!d->Init()) {
        LOG(LERROR, "disk init failed");
        return false;
    }

    disk_ = unique_ptr<Disk>(d);

    short port = stoi(get_conf("master.port"));
    auto accepter = shared_ptr<Handler>(new AcceptHandler(get_conf("master.ip"), port));
    if (accepter->fd < 0) {
        LOG(LERROR, "listen " << get_conf("master.ip") << ":" << port << "failed");
        return false;
    }

    int access = open(get_conf("log.access").c_str(), O_WRONLY|O_APPEND|O_CREAT, S_IWUSR);
    if (access < 0) {
        LOG(LERROR, "open access log file" << get_conf("log.access") << "failed");
        return false;
    }

    int wc = stoi(get_conf("worker.count"));
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


void Hornet::Stop()
{
    EventEngine::Stop();
    for (auto &w: workers_) {
        w->Stop();
    }
}


unique_ptr<Hornet> server;


void signal_handler(int sig)
{
    static bool s_handle_signal = false;
    LOG(LERROR, "signal_handler: " << sig);

    if (!s_handle_signal) {
        s_handle_signal = true;
        server->Stop();
    }
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
            || signal(SIGINT, signal_handler) == SIG_ERR) {
            LOG(LERROR, "setup signal failed");
            return 1;
        }

        return server->Start() ? 0 : 1;

    } catch (const exception & exc) {

        LOG(LERROR, exc.what());
        return 1;
    }
}
