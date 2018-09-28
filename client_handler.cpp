#include "client_handler.h"


const int STATUS_OK = 200;
const int STATUS_CREATED = 201;
const int STATUS_NOT_FOUND = 404;
const int STATUS_SERVER_ERROR = 500;


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

// version time time-cost method uri stat content-len error svr-ext client-ext
const char *LOG_FROMART = "%d %llu %llu %s %s %u %lu %s %s %s\n";

const char* RSP_TEMPLATE = "HTTP/1.1 %d %s\r\nContent-Length: 0\r\n\r\n";
const regex ARG_REGEX("(\\w+)=(\\w+)&?");
const regex HEADER_REGEX("(.+): (.+)\r\n");
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
    req_.phase = PH_READ_HEADER;

    req_.id = 0;
    req_.dir = 0;
    req_.state = 0;
    req_.method = 0;
    req_.header_len = 0;
    req_.write_len = 0;
    req_.start = 0;
    req_.content_len = 0;
    req_.send_len = 0;
    req_.recv_ = 0;

    req_.method_str = "-";
    req_.uri_str = "-";
    req_.client_ext = "-";
    req_.server_ext = "-";

    req_.error = "-";

    req_.args.clear();
    req_.headers.clear();

    req_.item.reset();
    req_.block.reset();

    send_.size = send_.offset = send_.processed = 0;
    recv_.size = recv_.offset = recv_.processed = 0;
}


void ClientHandler::getItem()
{
    req_.phase = PH_SEND_MEM;
    req_.state = STATUS_NOT_FOUND;

    if (disk_->Get(req_.dir, req_.id, req_.item, req_.block)) {
        req_.phase = PH_SEND_DISK;
        req_.state = STATUS_OK;
    }
}


void ClientHandler::addItem()
{
    if (disk_->Get(req_.dir, req_.id, req_.item, req_.block)) {
        req_.state = STATUS_OK;
        return;
    }

    if (req_.headers.find("Content-Length") == req_.args.end()) {
        req_.error = "NO_CONTENT_LENTH";
        return;
    }

    size_t cl = stoull(req_.headers["Content-Length"]);
    if (cl > stoull(get_conf("disk.block.size"))) {
        req_.error = "ITEM_TOO_BIG";
        return;
    }

    if (req_.args.find("expire") == req_.args.end()) {
        req_.args["expire"] = get_conf("item.default_expire");
    }

    stringstream buf;
    buf << "HTTP/1.1 200 OK\r\nServer: Hornet\r\n";
    for (auto& h: req_.headers) {
        buf << h.first << ": " << h.second << "\r\n";
    }
    buf << "\r\n";
    const string& header = buf.str();

    req_.item = shared_ptr<Item>(new Item());
    req_.item->putting = true;
    req_.item->expired = g_now + stoul(req_.args["expire"]);
    req_.item->header_size = header.size();
    req_.item->size = req_.item->header_size + cl;

    const char* err = parse_tags(req_.args, req_.item->tags);
    if (err != nullptr) {
        req_.error = err;
        return;
    }

    if (!disk_->Add(req_.dir, req_.id, req_.item, req_.block)) {
        req_.error = "ADD_DISK_FAILD";
        return;
    }

    if (!req_.block->Wirte(req_.item.get(), header.c_str(), header.size(), 0)) {
        req_.error = "WRITE_FAILD";
        return;
    }
    
    req_.write_len = req_.item->header_size;
    if (!req_.block->Wirte(req_.item.get(), recv_.data.get() + req_.header_len,
                       recv_.size - req_.header_len, req_.write_len)) {
        req_.error = "WRITE_FAILD";
        return;
    }

    req_.state = STATUS_CREATED;
    req_.phase = PH_READ_BODY;
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

    uint32_t n = disk_->Delete(req_.dir, req_.id, tags);
    req_.server_ext = to_string(n);
}



void ClientHandler::readHeader(Event* ev, EventEngine* engine)
{
    if (!ev->read) {
        return;
    }

    if (!read_buf(fd, recv_)) {
        req_.error = "READ_CLIENT_FAILD";
    }

    req_.start = g_now_ms;
    req_.recv_len += recv_.size;

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

    req_.phase = PH_SEND_MEM;
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


    // TODO: use a function
    const char* tmpstart;

    // args
    if (req_match.size() == 5 && req_match[4].length() != 0) {
        cmatch arg_match;
        for (tmpstart = req_match[4].first;
            regex_search(tmpstart, arg_match, ARG_REGEX);
            tmpstart = arg_match[0].second) {
            req_.args[arg_match[1].str()] = arg_match[2].str();
        }
    }

    // headers
    cmatch header_match;
    for (tmpstart = req_match[0].second; 
        regex_search(tmpstart, header_match, HEADER_REGEX);
        tmpstart = header_match[0].second) {
        req_.headers[header_match[1].str()] = header_match[2].str();
    }

    ev->read = false;
    ev->write = true;

    switch (req_.method) {
        case METHOD_GET:
            return getItem();

        case METHOD_PUT:
        case METHOD_POST:
            addItem();
            if (req_.error == nullptr && req_.state == STATUS_CREATED) {
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
        req_.recv_len += recv_.size;
        if (!req_.block->Wirte(req_.item.get(), recv_.data.get() + recv_.processed,
                        recv_.size - recv_.processed, req_.write_len)) {
            req_.error = "WRITE_FAILD";
            return;
        }

        if (recv_.size == 0)  {
            if (req_.write_len == req_.item->size) {
                req_.item->putting = false;
                req_.phase = PH_SEND_MEM;
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

    // 应该不属于这个函数
    if (send_.size == 0) {
        send_.size = snprintf(
            send_.data.get(), send_.capcity, RSP_TEMPLATE,
            req_.state, g_http_status[req_.state].c_str()
        );
    }

    if (!send_buf(fd, send_)){
        req_.error = "SEND_CLIENT_FAILD";
        return;
    }

    req_.write_len = send_.offset;
    ev->read = ev->write = ev->error = false;

    if (send_.offset == send_.size) {
        req_.phase = PH_FINISH;
    }
}


void ClientHandler::sendDisk(Event* ev, EventEngine* engine)
{
    if (!ev->write) {
        return;
    }

    if (!req_.block->Send(req_.item.get(), fd, req_.write_len)) {
        req_.error = "SEND_CLIENT_FAILD";
        return;
    }

    if (req_.write_len == req_.item->size) {
        req_.phase = PH_FINISH;
    }
}


void ClientHandler::finish(Event* ev, EventEngine* engine)
{
    ev->write = ev->read = ev->timer = false;

    if (recv_.size == 0 && req_.error != nullptr) {
        // client close connection after a transport
        return;
    }

    if (req_.headers.find("Client-Ext") != req_.headers.end()
        && req_.headers["Client-Ext"] != "") {
        req_.client_ext = req_.headers["Client-Ext"];
    }

    ssize_t n = snprintf(
        logger_->Buffer(), ACCESS_LOG_BUF, LOG_FROMART,
        VERSION, g_now, g_now_ms - req_.start,
        req_.method_str.c_str(),
        req_.uri_str.c_str(),
        req_.state,
        req_.content_len,
        req_.error == nullptr ? "-": req_.error,
        req_.server_ext.c_str(),
        req_.client_ext.c_str()
    );

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
            req_.phase = PH_FINISH;
        }

        switch (req_.phase) {
            case PH_IDLE:
                idle(ev, engine);
                break;

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
                LOG(LERROR, "unknown phase " << req_.phase);
                return false;
        }
    }

    return req_.error == nullptr;
}
