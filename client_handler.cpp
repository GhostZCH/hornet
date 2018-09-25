#include "client_handler.h"


const int STATUS_OK = 200;
const int STATUS_CREATED = 201;
const int STATUS_NOT_FOUND = 404;


pair<int, string> http_status[] = {
    {STATUS_OK, "OK"},
    {STATUS_CREATED, "Created"},
    {STATUS_NOT_FOUND, "Not Found"},
};


unordered_map<int, string> g_http_status(
    http_status,
    http_status + sizeof(http_status) / sizeof(pair<int, string>)
);


const int METHOD_GET = 1;
const int METHOD_PUT = 2;
const int METHOD_POST = 3;
const int METHOD_DELETE = 4;


pair<string, int> http_methods[] = {
    {"GET", METHOD_GET},
    {"PUT", METHOD_PUT},
    {"POST", METHOD_POST},
    {"DELETE", METHOD_DELETE},
};


unordered_map<string, int> g_http_methods(
    http_methods,
    http_methods + sizeof(http_methods) / sizeof(pair<string, int>)
);

// version time time-cost method uri stat content-len error 
const char *LOG_FROMART = "%d %llu %llu %s %s %u %lu %s\n";

const char* RSP_TEMPLATE = "HTTP/1.1 %d %s\r\nContent-Length: 0\r\n\r\n";
const regex ARG_REGEX("(\\w+)=(\\w+)&?");
const regex HEADER_REGEX("(.+): (\\w+)\r\n");
const regex REQ_LINE_REGEX("^(GET|POST|PUT|DELETE) /(\\d+)/(\\d+)\\??""(.*) HTTP/1.1\r\n");


const char* parse_tags(const map<string, string>& args, uint16_t* tags)
{
    for (int i = 0; i < TAG_LIMIT; i++) {
        tags[i] = 0;
        auto iter = args.find("tag" + to_string(i));
        if (iter != args.end()) {
            int temp = stoi(iter->second);
            if (temp < 0 || temp > 65534) {
                return "tags not in range [0, 65534]";
            }
            tags[i] = temp;
        }
    }
    return nullptr;
}


bool read_buf(int fd, Buffer& buf)
{
    while(buf.size < buf.capcity) {
        int n = read(fd, buf.data.get() + buf.size, buf.capcity - buf.size);
        if (n > 0) {
            buf.size += n;
        } else {
            return n < 0 && errno == EAGAIN;
        }
    }
    return true;
}


bool send_buf(int fd, Buffer& buf)
{
    while(buf.offset < buf.size) {
        int n = write(fd, buf.data.get() + buf.offset, buf.size - buf.offset);
        if (n > 0) {
            buf.offset += n;
        } else {
            return n < 0 && errno == EAGAIN;
        }
    }
    return true;
}



ClientHandler::ClientHandler()
    :Handler()
{
    timeout_ = 0;

    send_.capcity = stoull(get_conf("request.send_buf"));
    recv_.capcity = stoull(get_conf("request.recv_buf"));
    send_.data = unique_ptr<char []>(new char[send_.capcity]);
    recv_.data = unique_ptr<char []>(new char[recv_.capcity]);

    reset();
}


bool ClientHandler::Init(EventEngine* engine)
{
    disk_ = (Disk *)engine->context["disk"];
    logger_ = (AccessLog *)engine->context["access"];

    auto h = shared_ptr<Handler>(this);
    timeout_ = g_now + stoull(get_conf("request.timeout"));
    return engine->AddHandler(h) && engine->AddEpollEvent(fd) && engine->AddTimer(fd, timeout_, 0);
}


bool ClientHandler::Close(EventEngine* engine)
{
    if (!engine->DelTimer(fd, timeout_, 0)) {
        return false;
    }

    return engine->DelEpollEvent(fd) && engine->DelHandler(fd);
}


void ClientHandler::reset()
{
    phase_ = PH_READ_HEADER;

    req_.id = 0;
    req_.dir = 0;
    req_.state = 0;
    req_.method = 0;
    req_.header_len = 0;
    req_.write_len = 0;

    req_.method_str = "-";
    req_.uri_str = "-";
    req_.error = nullptr;

    req_.args.clear();
    req_.headers.clear();;

    item_.reset();
    block_.reset();

    send_.size = send_.offset = send_.processed = 0;
    recv_.size = recv_.offset = recv_.processed = 0;
}


void ClientHandler::getItem()
{
    phase_ = PH_SEND_MEM;
    req_.state = STATUS_NOT_FOUND;

    if (disk_->Get(req_.dir, req_.id, item_, block_)) {
        phase_ = PH_SEND_DISK;
        req_.state = STATUS_OK;
    }
}


void ClientHandler::addItem()
{
    if (!disk_->Get(req_.dir, req_.id, item_, block_)) {
        req_.state = STATUS_OK;
        return;
    }

    if (req_.args.find("Content-Length") == req_.args.end()) {
        req_.error = "NO_CONTENT_LENTH";
        return;
    }

    size_t cl = stoull(req_.args["Content-Length"]);
    if (cl > stoull(get_conf("disk.block.size"))) {
        req_.error = "ITEM_TOO_BIG";
        return;
    }

    stringstream buf("HTTP/1.1 200 OK\r\nServer: Hornet\r\n");
    for (auto& h: req_.headers) {
        buf << h.first << ": " << h.second << "\r\n";
    }
    buf << "\r\n";
    const string& header = buf.str();

    auto item = shared_ptr<Item>(new Item());
    item->putting = true;
    item->expired = stoul(req_.args["expire"]);
    item->header_size = header.size();
    item->size = item->header_size + cl;

    const char* err = parse_tags(req_.args, item->tags);
    if (err != nullptr) {
        req_.error = err;
        return;
    }

    if (disk_->Add(req_.dir, req_.id, item, block_)) {
        req_.error = "ADD_DISK_FAILD";
        return;
    }

    if (!block_->Wirte(item_.get(), header.c_str(), header.size(), 0)) {
        req_.error = "WRITE_FAILD";
        return;
    }
    
    req_.write_len = item_->header_size;
    if (!block_->Wirte(item_.get(), recv_.data.get() + req_.header_len,
                       recv_.size - req_.header_len, req_.write_len)) {
        req_.error = "WRITE_FAILD";
        return;
    }

    phase_ = PH_READ_BODY;
    recv_.processed = req_.header_len;
}


void ClientHandler::delItem()
{
    uint16_t tags[TAG_LIMIT];

    const char* err = parse_tags(req_.args, tags);
    if (err != nullptr) {
        req_.error = err;
        return;
    }

    disk_->Delete(req_.dir, req_.id, tags);
}


void ClientHandler::readHeader(Event* ev, EventEngine* engine)
{
    if (!ev->read) {
        return;
    }

    if (!read_buf(fd, recv_)) {
        req_.error = "READ_CLIENT_FAILD";
        return;
    }

    char *start = recv_.data.get() + recv_.processed;
    char *end = (char *)memmem(start, recv_.size - recv_.processed, "\r\n\r\n", 4);

    recv_.processed = recv_.size;

    if (end == nullptr) {
        if (recv_.size == recv_.capcity) {
            req_.error = "HEADER_TOO_LARGE";
            return;
        }
        return;
    }

    phase_ = PH_SEND_MEM;
    req_.header_len = end - recv_.data.get() + 4;

    cmatch req_match;
	if (!regex_search(recv_.data.get(), req_match, REQ_LINE_REGEX)) {
        req_.error = "REQ_LINE_ERR";
        return;
    }

    req_.method_str = req_match[1].str();
    req_.uri_str = string(req_match[2].first, req_match[4].second);

    // method
    req_.method = g_http_methods[req_.method_str];

    req_.dir = stoull(req_match[2].str());
    req_.id = stoull(req_match[3].str());

    if (req_.method != METHOD_GET) {
        const char* start;
        // args
        if (req_match.size() == 5 && req_match[4].length() != 0) {
            cmatch arg_match;
            for (start = req_match[4].first;
                regex_search(start, arg_match, ARG_REGEX);
                start = arg_match[0].second) {
                req_.args[arg_match[1].str()] = arg_match[2].str();
            }
        }

        // headers
        cmatch header_match;
        for (req_match[0].second; 
            regex_search(start, header_match, HEADER_REGEX);
            start = header_match[0].second) {
            req_.headers[header_match[1].str()] = header_match[2].str();
        }
    }

    ev->read = false;
    ev->write = true;

    switch (req_.method) {
        case METHOD_GET:
            return getItem();

        case METHOD_PUT:
        case METHOD_POST:
            addItem();
            if (req_.error == nullptr) {
                ev->write = false;
                ev->read = true;
            }
            return;

        case METHOD_DELETE:
            return delItem();
    }
}


void ClientHandler::readBody(Event* ev, EventEngine* engine)
{
    if (!ev->read) {
        return;
    }

    while (read_buf(fd, recv_)) {
        if (!block_->Wirte(item_.get(), recv_.data.get() + recv_.processed,
                        recv_.size - recv_.processed, req_.write_len)) {
            req_.error = "WRITE_FAILD";
            return;
        }

        if (recv_.size == 0)  {
            if (req_.write_len == item_->size) {
                phase_ = PH_SEND_MEM;
                ev->read = false;
                ev->write = true;
            }
            return;
        }

        req_.write_len += recv_.size - recv_.processed;
        recv_.processed = recv_.size = 0;
    }

    req_.error = "READ_CLIENT_FAILD";
}


void ClientHandler::sendMem(Event* ev, EventEngine* engine)
{
    if (!ev->write) {
        return;
    }

    if (send_.size == 0) {
        send_.size = snprintf(send_.data.get(), send_.capcity, RSP_TEMPLATE,
                              req_.state, g_http_status[req_.state].c_str());
    }

    if (!send_buf(fd, send_)){
        req_.error = "SEND_CLIENT_FAILD";
        return;
    }
    ev->read = ev->write = ev->error = false;

    if (send_.offset == send_.size) {
        phase_ = PH_FINISH;
    }
}


void ClientHandler::sendDisk(Event* ev, EventEngine* engine)
{
    if (!ev->write) {
        return;
    }

    if (!block_->Send(item_.get(), fd, req_.write_len)) {
        req_.error = "SEND_CLIENT_FAILD";
        return;
    }

    if (req_.write_len == item_->size) {
        phase_ = PH_FINISH;
    }
}


void ClientHandler::finish(Event* ev, EventEngine* engine)
{
    ev->write = ev->read = ev->timer = false;

    ssize_t n = snprintf(logger_->Buffer(), ACCESS_LOG_BUF,
                        LOG_FROMART, VERSION,
                        g_now, g_now_ms - req_.start,
                        req_.method_str.c_str(), req_.uri_str.c_str(),
                        req_.state, req_.content_len,
                        req_.error == nullptr ? "-": req_.error);

    logger_->Log(logger_->Buffer(), n);

    if (req_.error != nullptr) {
        return;
    }

    reset();

    if (!engine->DelTimer(fd, timeout_, 0)) {
        LOG(LERROR, "ClientHandler::finish DelTimer error");
        req_.error = "SYS_ERROR";
    }

    timeout_ += g_now + stoull(get_conf("request.timeout"));
    if (!engine->AddTimer(fd, timeout_, 0)) {
        LOG(LERROR, "ClientHandler::finish AddTimer error");
        req_.error = "SYS_ERROR";
    }
}


void ClientHandler::timeout(Event* ev, EventEngine* engine)
{
    if (!ev->timer) {
        return;
    }

    req_.error = "TIMEOUT";
}

bool ClientHandler::Handle(Event* ev, EventEngine* engine)
{
    while (ev->write || ev->read || ev->timer) {
        timeout(ev, engine);

        if (req_.error) {
            phase_ = PH_FINISH;
        }

        switch (phase_) {
            case PH_READ_HEADER:
                readHeader(ev, engine);
                break;

            case PH_READ_BODY:
                readBody(ev, engine);
                break;

            case PH_SEND_MEM: 
                sendMem(ev, engine);
                break;

            case PH_SEND_DISK:
                sendDisk(ev, engine);
                break;

            case PH_FINISH:
                finish(ev, engine);
                break;

            default:
                LOG(LERROR, "unknown phase " << phase_);
                return false;
        }
    }

    return req_.error == nullptr;
}
