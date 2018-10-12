#include "client_handler.h"


ClientHandler::ClientHandler()
    :Handler()
{
    timeout_ = 0;
}


void ClientHandler::Init(EventEngine* engine)
{

    auto h = shared_ptr<Handler>(this);
    timeout_ = g_now + stoull(get_conf("request.timeout"));

    engine->AddHandler(h);
    engine->AddEpollEvent(fd);
    engine->AddTimer(fd, timeout_, 0);
}


void ClientHandler::Close(EventEngine* engine)
{
    engine->DelTimer(fd, timeout_, 0);
    engine->DelEpollEvent(fd);
    engine->DelHandler(fd);
}


void ClientHandler::Handle(Event* ev, EventEngine* engine)
{
    if (!req_) {
        if (ev->timer || ev->error) {
            throw ReqError("client close", __FILE__, __LINE__);
        }

        char tmp;
        if (recv(fd, &tmp, 1, MSG_PEEK) != 1) {
            throw ReqError("client close", __FILE__, __LINE__);
        }

        auto disk = static_cast<Disk*>(engine->context["disk"]);
        auto logger = static_cast<AccessLog*>(engine->context["access"]);
        req_ = unique_ptr<Request>(new Request(fd, disk, logger));
    }

    if (ev->timer) {
        req_->Timeout();
    }

    if (ev->error) {
        req_->Error();
    }

    bool go = true;
    while (go) {
        switch (req_->Phase()) {
            case PH_READ_HEADER:
                go = req_->ReadHeader();
                break;
            case PH_READ_BODY:
                go = req_->ReadBody();
                break;
            case PH_SEND_RSP:
                go =req_->SendResponse();
                break;
            case PH_SEND_CACHE:
                go = req_->SendCache();
                break;
            case PH_FINISH:
                go = req_->Finish();
                req_.reset();
                break;
            default:
                throw SvrError("unknown phase " + to_string(req_->Phase()), __FILE__, __LINE__);
        }
    }
}
