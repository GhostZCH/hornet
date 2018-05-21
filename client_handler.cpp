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


ClientHandler::ClientHandler(Disk* disk)
{
    disk_ = disk;
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


bool ClientHandler::GetSpecialHeader()
{
    const char* temp = "HTTP/1.1 %s\r\nConnection: keep-alive\r\nContent-Length: 0\r\n\r\n";

    buf_size_ = snprintf(buf_.get(), buf_capacity_, temp, g_http_status[state_].c_str());

    return true;
}


bool ClientHandler::ProcessInput()
{
    if (strstr(buf_.get() + process_len_, "\r\n\r\n") != nullptr) {
        reading_ = false;
        process_len_ = 0;

        char *end = strchr(buf_.get(), ' ');

        if (end == nullptr) {
            state_ = HTTP_BAD_REQUEST;
            return true;
        }

        *end = '\0';

        auto method_iter = g_http_methods.find(buf_.get());
        if (method_iter == g_http_methods.end()) {
            state_ = HTTP_BAD_REQUEST;
            return true;
        }

        method_ = method_iter->second;

        end++; // ' '
        Key dir;
        Key id;

        dir.Load(end + 1); // '/'
        id.Load(end + 1 + KEY_CHAR_SIZE + 1); // /dir/id

        if (method_ == HTTP_GET) {
            item_ = disk_->Get(dir, id);

            if (item_ == nullptr) {
                state_ = HTTP_NOT_FOUND;
                return true;
            }

            state_ = HTTP_OK;
            process_len_ = 0;

            return true;
        }

    } else {

        process_len_ = buf_size_;

        if (process_len_ == buf_capacity_) {
            state_ = HTTP_HEADER_TOO_LARGE; //TODO send special respose
            return true;
        }
    }

    return true;
}


bool ClientHandler::Read()
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

    return ProcessInput();
}


bool ClientHandler::Write()
{
    while (true) {
        ssize_t n = write(fd, buf_.get() + process_len_, buf_size_ - process_len_);
        if (n > 0) {
            process_len_ += n;
            if (process_len_ == buf_size_) {
                //reset 
                process_len_ = 0;
                buf_size_ = 0;
                reading_ = true;

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
        if (!Read()) {
            return false;
        }

        if (!reading_) {
            ev->write = true;

            if (state_ >= HTTP_BAD_REQUEST) {
                if (!GetSpecialHeader()) {
                    return false;
                }
            }
        }
    }

    if (ev->write && !reading_) {
        if (!Write()) {
            return false;
        }
    }

    return true;
}

