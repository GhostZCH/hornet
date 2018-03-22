#include "hornet.h"


int main()
{
    Master master("127.0.0.1", 8080, 1u);
    master.Forever();

    return 0;
}