#include "hornet.h"

// TODO load conf
const int worker_count = 4;

unique_ptr<Master> g_master;


void signal_handler(int sig)
{
    (*g_master).Stop();
}


int main(int argc, char* argv[])
{
    g_master = unique_ptr<Master>(new Master());

    vector<unique_ptr<thread>> workers;
    for (int i = 0; i < worker_count; i++) {
        Worker *worker = new Worker(i);
        (*g_master).AddWorker(worker);
        workers.push_back(unique_ptr<thread>(new thread(*worker)));
    }

    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    (*g_master).Forever();

    for (int i = 0; i < worker_count; i++) {
        (*workers[i]).join();
    }

    return 0;
}