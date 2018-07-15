#include "worker.h"


Worker::Worker(int id, Disk *disk): EventEngine()
{
    id_ = id;

    logger(LOG_WARN, "worker[" << id_ << "] start");
}
