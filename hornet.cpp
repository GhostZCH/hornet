#include "hornet.h"
#include "tool.h"
#include "master.h"
#include "accept_handler.h"


unique_ptr<Master> master;


void signal_handler(int sig)
{
    static bool s_handle_signal = false;
    logger(LOG_ERROR, "signal_handler: " << sig);

    if (!s_handle_signal) {
        s_handle_signal = true;
        master->Stop();
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
            logger(LOG_ERROR, "load_conf failed");
            return 1;
        }

        ofstream errlog = ofstream(g_config["log.error"], ios_base::app);
        if (!errlog.is_open() || !set_logger(g_config["log.level"], &errlog)) {
            logger(LOG_ERROR, "open errlog failed");
            return 1;
        }

        master = unique_ptr<Master>(new Master());

        if (signal(SIGTERM, signal_handler) == SIG_ERR || signal(SIGINT, signal_handler) == SIG_ERR) {
            logger(LOG_ERROR, "setup signal failed");
            return 1;
        }

        return master->Start() ? 0 : 1;

    } catch (const exception & exc) {

        logger(LOG_ERROR, exc.what());
        return 1;
    }
}
