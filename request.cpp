#include "request.h"


pair<int, string> http_status[] = {
    {STATUS_OK, "200 OK"},
    {STATUS_CREATED, "204 Created"},
    {STATUS_NOT_FOUND, "404 Not Found"},
};


unordered_map<int, string> g_http_status(
    http_status,
    http_status + sizeof(http_status) / sizeof(pair<int, string>)
);


// version time time-cost method uri stat recvlen sendlen error svr-ext client-ext
const char *LOG_FROMART = "%d %llu %llu %s %s %u %lu %lu %s %s %s\n";
const char* RSP_TEMPLATE = "HTTP/1.1 %s\r\nServer: Hornet\r\nContent-Length: 0\r\n\r\n";

const regex ARG_REGEX("(\\w+)=(\\w+)&?");
const regex HEADER_REGEX("(.+): (.+)\r\n");
const regex REQ_LINE_REGEX("^(GET|POST|DELETE) /(\\d+)/(\\d+)\\??(.*) HTTP/1.1\r\n");


Request::Request(int fd, Disk *d, AccessLog* log)
{
    fd_ = fd;
    disk_ = d;
    log_ = log;

    start_ = g_now_ms;
    phase_ = PH_READ_HEADER;

    id_ = 0;
    dir_ = 0;
    state_ = STATUS_OK;
    header_len_ = 0;

    uri_ = "-";
    method_ = "-";

    error_ = "-";

    client_ext_ = "-";
    server_ext_ = "-";

    size_t size = stoull(get_conf("request.send_buf"));
    recv_ = unique_ptr<Buffer>(new MemBuffer(size));
}


bool Request::ReadHeader()
{
    if (!recv_->Recv(fd_)) {
        error_ = "READ-ERROR";
        return false;
    }

    auto recv = dynamic_cast<MemBuffer *>(recv_.get());
    char *end = (char *)memmem(recv->Get(), recv_->recved, "\r\n\r\n", 4);
    if (end == nullptr) {
        if (recv->recved == recv->size) {
            error_ = "HEADER-TOO-LARGE";
            return false;
        }
        return true;
    }

    header_len_ = end - recv->Get();

    const char *args, *headers;
    if (!parseReqLine(args, headers) || !parseArgs(args) || !parseHeaders(headers)) {
        error_ = "BAD-REQUEST";
        return false;
    }

    if (method_ == "GET") {
        return getItem();
    }

    if (method_ == "POST") {
        return addItem();
    }
    
    return delItem();
}


bool Request::ReadBody()
{
    // TODO
}


bool Request::SendResponse()
{

}


bool Request::SendCache()
{

}


bool Request::Finish()
{
    if (recv_->size == 0) {
        return;
    }

    if (headers_.find("Client-Ext") != headers_.end()
        && headers_["Client-Ext"] != "") {
        client_ext_ = headers_["Client-Ext"];
    }

    ssize_t n = snprintf(
        log_->Buffer(), ACCESS_LOG_BUF, LOG_FROMART,
        VERSION, g_now, g_now_ms - start_,
        method_.c_str(),
        uri_.c_str(),
        state_,
        0, //TODO recv len
        0, //TODO send len
        error_.c_str(),
        server_ext_.c_str(),
        client_ext_.c_str()
    );

    return log_->Log(log_->Buffer(), n);
}


bool Request::Timeout()
{
    error_ = "TIME-OUT";
    phase_ = PH_FINISH;
    return true;
}


bool Request::Error()
{
    error_ = "CONNECTION-ERR";
    phase_ = PH_FINISH;
    return true;
}


bool Request::parseReqLine(const char* &args, const char* &headers)
{
    auto recv = dynamic_cast<MemBuffer *>(recv_.get());

    cmatch match;
	if (!regex_search(recv->Get(), match, REQ_LINE_REGEX)) {
        return false;
    }

    method_ = match[1].str();
    uri_ = string(match[2].first, match[4].second);

    id_ = stoull(match[2].str());
    dir_ = stoull(match[3].str());

    args = match[4].first;
    headers = match[0].second;

    return true;
}


bool Request::parseHeaders(const char *start)
{
    cmatch match;

    while(regex_search(start, match, HEADER_REGEX)){
        headers_[match[1].str()] = match[2].str();
        start = match[0].second;
    }

    return true;
}


bool Request::parseArgs(const char *start)
{
    cmatch match;

    while(regex_search(start, match, ARG_REGEX)) {
        args_[match[1].str()] = match[2].str();
        start = match[0].second;
    }

    return true;
}


bool Request::parseTags(uint16_t tags[])
{
    for (int i = 0; i < TAG_LIMIT; i++) {
        tags[i] = 0;
        auto iter = args_.find("tag" + to_string(i));
        if (iter != args_.end()) {
            int t = stoi(iter->second);
            if (t < 0 || t > 65534) {
                error_ = "TAG-ERROR";
                return false;
            }
            tags[i] = t;
        }
    }

    return true;
}


bool Request::getItem()
{
    shared_ptr<Item> item;
    shared_ptr<Block> block;
    if (disk_->Get(dir_, id_, item, block)) {
        send_ = unique_ptr<Buffer>(new FileBuffer(block->Fd(), item->pos, item->size));
        phase_ = PH_SEND_CACHE;
        state_ = STATUS_OK;
    } else {
        phase_ = PH_SEND_RSP;
        state_ = STATUS_NOT_FOUND;
    }

    return true;
}


bool Request::addItem()
{
    shared_ptr<Item> item;
    shared_ptr<Block> block;

    if (disk_->Get(dir_, id_, item, block)) {
        state_ = STATUS_OK;
        phase_ = PH_SEND_RSP;
        return;
    }

    if (headers_.find("Content-Length") == headers_.end()) {
        error_ = "NO_CONTENT_LENTH";
        return false;
    }

    size_t cl = stoull(headers_["Content-Length"]);
    if (cl > stoull(get_conf("disk.block.size"))) {
        error_ = "ITEM_TOO_BIG";
        return false;
    }

    if (headers_.find("expire") == headers_.end()) {
        headers_["expire"] = get_conf("item.default_expire");
    }

    stringstream buf;
    buf << "HTTP/1.1 200 OK\r\nServer: Hornet\r\n";
    for (auto& h: headers_) {
        buf << h.first << ": " << h.second << "\r\n";
    }
    buf << "\r\n";
    const string& header = buf.str();

    item = shared_ptr<Item>(new Item());
    item->putting = true;
    item->expired = g_now + stoul(args_["expire"]);
    item->header_size = header.size();
    item->size = item->header_size + cl;

    if (!parseTags(item->tags)) {
        return false;
    }

    if (!disk_->Add(dir_, id_, item, block)) {
        error_ = "ADD_DISK_FAILD";
        return false;
    }

    auto file = new FileBuffer(block->Fd(), item->pos, item->size);
    auto mem = dynamic_cast<MemBuffer*>(recv_.get());

    // write headers and remain body
    if (!file->Write(header.c_str(), header.size())
        || !file->Write(mem->Get() + header_len_, mem->recved - header_len_)) {
        error_ = "WRITE_FAILD";
        delete file;
        return false;
    }

    // relead membuf, set filebuf
    recv_.reset();
    recv_ = unique_ptr<Buffer>(file);
    phase_ = PH_READ_BODY;

    return true;
}


bool Request::delItem()
{
    uint16_t tags[TAG_LIMIT];

    if (!parseTags(tags)) {
        return false;
    }

    uint32_t n = disk_->Delete(dir_, id_, tags);
    server_ext_ = to_string(n);
    return true;
}
