#include "client_handler.h"


const int HTTP_OK = 200;
const int HTTP_CREATED = 201;
const int HTTP_BAD_REQUEST = 400;
const int HTTP_NOT_FOUND = 404;
const int HTTP_LENGTH_REQUIRED = 411;
const int HTTP_REQUEST_ENTITY_TOO_LARGE = 413;
const int HTTP_HEADER_TOO_LARGE = 494;
const int HTTP_INTERNAL_SERVER_ERROR = 500;


pair<int, string> http_status[] = {
    {HTTP_OK, "200 OK"},
    {HTTP_CREATED, "201 Created"},
    {HTTP_BAD_REQUEST, "400 Bad Request"},
    {HTTP_NOT_FOUND, "404 Not Found"},
    {HTTP_LENGTH_REQUIRED, "411 Length Required"},
    {HTTP_REQUEST_ENTITY_TOO_LARGE, "413 Request Entity Too Large"},
    {HTTP_HEADER_TOO_LARGE, "494 Request Header Too Large"},
    {HTTP_INTERNAL_SERVER_ERROR, "500 Internal Server Error"},
};


unordered_map<int, string> g_http_status(
    http_status,
    http_status + sizeof(http_status) / sizeof(pair<int, string>));


const int HTTP_GET = 1;
const int HTTP_POST = 2;
const int HTTP_PUT = 3;
const int HTTP_DELETE = 4;


pair<string, int> http_methods[] = {
    {"GET", HTTP_GET},
    {"POST", HTTP_POST},
    {"PUT", HTTP_PUT},
    {"DELETE", HTTP_DELETE},
};


unordered_map<string, int> g_http_methods(
    http_methods,
    http_methods + sizeof(http_methods) / sizeof(pair<string, int>));


const char *RES_LINE = "HTTP/1.1 200 OK\r\nServer: Hornet";
const size_t RES_LINE_LEN = strlen(RES_LINE);

const regex REQ_LINE_REGEX("(GET|POST|PUT) /([0-9a-f]{32})/([0-9a-f]{32})\\\?(?:(\\w+)=(\\d+)&)*(?:(\\w+)=(\\d+)) HTTP/1.1");
const size_t REQ_LINE_CAP_MIN = 4;


ClientHandler::ClientHandler(Disk* disk, size_t buf_cap)
{
    disk_ = disk;
    buf_capacity_ = buf_cap;
}


bool ClientHandler::Init(EventEngine* engine)
{
    if (!engine->AddHandler(this) || !engine->AddEpollEvent(fd)) {
        return false;
    }

    buf_ = unique_ptr<char []>(new char[buf_capacity_]);

    return true;
}


bool ClientHandler::Close(EventEngine* engine)
{
    if (!engine->DelEpollEvent(fd) || !engine->DelHandler(fd)) {
        return false;
    }

    close(fd);
    return true;
}


bool ClientHandler::setRspHeader()
{
    const char* rsp_template = "HTTP/1.1 %s\r\nContent-Length: 0\r\n\r\n";

    buf_size_ = snprintf(buf_.get(),
                        buf_capacity_,
                        rsp_template,
                        g_http_status[state_].c_str());

    send_disk_ = false;

    return true;
}


bool ClientHandler::processReqLine(char *header_end)
{
    cmatch match;

    char *req_line_end = strstr(buf_.get(), "\r\n");

    *req_line_end = '\0';
    
    if (!regex_match(buf_.get(), match, REQ_LINE_REGEX)
        || match.size() < REQ_LINE_CAP_MIN
        || (match.size() - REQ_LINE_CAP_MIN) % 2 != 0) {

        state_ = HTTP_BAD_REQUEST;
        return setRspHeader();
    }

    method_ = g_http_methods[match[1].str()];

    Key dir(match[2].first);
    Key id(match[3].first);

    if (method_ == HTTP_GET) {
        item_ = disk_->Get(dir, id);

        if (item_ == nullptr) {
            state_ = HTTP_NOT_FOUND;
            return setRspHeader();

        }

        state_ = HTTP_OK;
        send_disk_ = true;
        return true;
    }

    for (size_t i = REQ_LINE_CAP_MIN; i < match.size(); i += 2) {
        args_[match[i]] = atoll(match[i+1].first);
    }

    if (method_ == HTTP_POST || method_ == HTTP_PUT) {
        item_ = disk_->Get(dir, id);

        if (item_ != nullptr) {
            state_ = HTTP_OK;
            return setRspHeader();
        }

        char *start = req_line_end - RES_LINE_LEN;
        memcpy(start, RES_LINE, RES_LINE_LEN);
        *req_line_end = '\r';

        Item temp = {0};

        temp.putting = 1;
        temp.use = 1;
        temp.expired = args_["expire"];
        temp.header_size = temp.size = header_end - start + 4;

        item_ = disk_->Add(dir, id, temp);
        if (item_ == nullptr) {
            state_ = HTTP_INTERNAL_SERVER_ERROR;
            return false;
        }

        disk_->Wirte(item_, start);
        state_ = HTTP_CREATED;
        item_->putting = 0;
        item_->use = 0;
        return setRspHeader();
    }

    state_ = 400;
    setRspHeader();
    return true;
}


bool ClientHandler::handleRead()
{
    while (true) {
        ssize_t n = read(fd, buf_.get() + buf_size_, buf_capacity_ - buf_size_);
        if (n > 0) {
            buf_size_ += n;
            continue;
        }

        if (n < 0 && errno == EAGAIN) {
            break;
        }

        return false;
    }

    char *header_end = strstr(buf_.get() + process_len_, "\r\n\r\n");

    if (header_end != nullptr) {
        reading_ = false;
        process_len_ = 0;
        buf_size_ = 0;

        return processReqLine(header_end);

    } else {

        process_len_ = buf_size_;

        if (process_len_ == buf_capacity_) {
            state_ = HTTP_HEADER_TOO_LARGE; //TODO send special respose
            return false;
        }

        return true;
    }
}


bool ClientHandler::handleWrite()
{
    while (true) {
        ssize_t n;

        if (send_disk_) {
            n = disk_->Send(item_, fd, process_len_);
        } else {
            n = write(fd, buf_.get() + process_len_, buf_size_ - process_len_);
        }

        if (n > 0) {
            process_len_ += n;
            if (process_len_ == (send_disk_ ? item_->size : buf_size_)) {
                //reset
                process_len_ = 0;
                buf_size_ = 0;
                reading_ = true;
                method_ = 0;
                state_ = 0;
                args_.clear();

                return true;
            }
            continue;
        }

        if (n < 0 && errno == EAGAIN) {
            break;
        }

        return false;
    }

    return true;
}


bool ClientHandler::Handle(Event* ev, EventEngine* engine)
{
    if (ev->read && reading_) {
        if (!handleRead()) {
            return false;
        }

        if (!reading_) {
            ev->write = true;
        }
    }

    if (ev->write && !reading_) {
        if (!handleWrite()) {
            return false;
        }
    }

    return true;
}

