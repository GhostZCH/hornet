#include "hornet.h"


bool GetMaster()
{
    Disk *disk = new Disk("/tmp/", 100, time(nullptr));
    Master *master = new Master(disk, new EventEngine());
}


int main()
{
    

    return 0;
}