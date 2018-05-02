#include "hornet.h"
#include "process.h"

int main(int argc, char* argv[])
{
    auto master = unique_ptr<Master>(new Master());

    if (!master->Init()) {
        return 1;
    }

    bool ok = master->Forever();

    return ok ? 0 : 1;

}