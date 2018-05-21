#include "hornet.h"
#include "process.h"
#include "disk.h"


void init_test_conf(map<string, string> &conf)
{
    conf["master.port"] = "1691";
    conf["master.ip"] = "0.0.0.0";

    conf["worker.count"] = "4";

    conf["disk.block.count"] = "4";
    conf["disk.block.size"] = to_string(1024*32); // 32M
    conf["disk.path"] = "/tmp/";
}


int main(int argc, char* argv[])
{
    unique_ptr<Master> master;

    try {
        map<string, string> conf;
        init_test_conf(conf);

        master = unique_ptr<Master>(new Master(conf));

        if (!master->Init()) {
            return 1;
        }

        return master->Forever() ? 0 : 1;
    } catch (const std::exception & exc) {
        cout << exc.what() << endl;
        return 1;
    }
}
