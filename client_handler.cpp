#include "client_handler.h"


BufferBlock::BufferBlock(size_t size)
{
    this->size = size;
}


BufferBlock::~BufferBlock()
{
    delete [] start;
}


bool BufferBlock::Init()
{
    start = new char[size];
    if (start == nullptr) {
        return false;
    }

    end = start + size;
    pos = start;

    return true;
}

void BufferBlock::Reset()
{
    end = start + size;
    pos = start;
}


bool ClientHandler::Init(EventEngine* engine)
{
    if (!engine->AddHandler(this) || !engine->AddEpollEvent(fd)) {
        return false;
    }

    pool_ = new Pool(1024*1024*100);
    if (pool_ == nullptr) {
        return false;
    }

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


bool ClientHandler::Handle(Event* ev, EventEngine* engine)
{
    const char *test = "HTTP/1.1 200 OK\r\nConnection: keep-alive\r\nContent-Length: 6\r\n\r\nhornet";

    if (ev->read && reading_) {

        while (true) {
            ssize_t n = read(fd, in_buf_->pos, in_buf_->end - in_buf_->pos);

            if (n > 0) {
                in_buf_->pos += n;
                continue;
            }

            if (n < 0 && errno == EAGAIN) {
                break;
            }

            return false;
        }

        if (strstr(in_buf_->start, "\r\n\r\n") != nullptr) {
            reading_ = false;
            ev->write = true;

            out_buf_->end = out_buf_->start + strlen(test);
            memcpy(out_buf_->start, test, out_buf_->end - out_buf_->start);
        }
    }

    if (ev->write && !reading_) {
        while (true) {

            ssize_t n = write(fd, out_buf_->pos, out_buf_->end - out_buf_->pos);

            if (n > 0) {
                out_buf_->pos += n;

                if (out_buf_->pos == out_buf_->end) {
                    //reset 
                    in_buf_->pos = in_buf_->start;
                    out_buf_->pos = out_buf_->start;
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
    }

    return true;
}

