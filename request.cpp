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
    arg_ = "-";
    method_ = "-";

    error = "-";

    client_ext = "-";
    server_ext = "-";

    size_t size = stoull(get_conf("request.send_buf"))
    recv_buf_ = unique_ptr<Buffer>(new MemBuffer(size));
}


bool Request::ReadHeader()
{
    if (!recv_buf_.Read(fd_)) {
        error_ = "READ-ERROR";
        return false;
    }

    char *end = (char *)memmem(recv_buf_->Get(), recv_buf_->recved, "\r\n\r\n", 4);
    if (end == nullptr) {
        if (recv_buf_->recved == recv_buf_.size) {
            error_ = "HEADER-TOO-LARGE";
            return false;
        }
        return true;
    }

    header_len_ = end - recv_buf_->Get();

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

}


bool Request::SendResponse()
{

}


bool Request::SendCache()
{

}


bool Request::Finish()
{
    if (recv_.size == 0) {
        return;
    }

    if (headers_.find("Client-Ext") != headers_.end()
        && headers_["Client-Ext"] != "") {
        req_.client_ext = headers_["Client-Ext"];
    }

    ssize_t n = snprintf(
        logger_->Buffer(), ACCESS_LOG_BUF, LOG_FROMART,
        VERSION, g_now, g_now_ms - start_,
        method_.c_str(),
        uri_str_.c_str(),
        state_,
        0, //TODO recv len
        0, //TODO send len
        error_.c_str(),
        server_ext.c_str(),
        client_ext.c_str()
    );

    return logger_->Log(logger_->Buffer(), n);
}


bool Request::Timeout()
{
    error_ = "TIME-OUT";
    phase_ = PH_FINISH;
    return true;
}


bool Request::Error()
{
    error_ = "CONNECTION-ERR"
    phase_ = PH_FINISH;
    return true;
}


bool Request::parseReqLine(const char* &args, const char* &headers)
{
    cmatch match;
	if (!regex_search(recv_->Get(), match, REQ_LINE_REGEX)) {
        return false;
    }

    method_ = match[1].str();
    uri_ = string(req_match[2].first, req_match[4].second);

    id_ = stoull(match[2].str());
    dir_ = stoull(match[3].str());

    args = match[4].first;
    headers = req_match[0].second;

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

}


bool Request::addItem()
{

}


bool Request::delItem()
{
    uint16_t tags[TAG_LIMIT];

    if (!parseTags(tags)) {
        return false;
    }

    uint32_t n = disk_->Delete(req_.dir, req_.id, tags);
    req_.server_ext = to_string(n);
}
