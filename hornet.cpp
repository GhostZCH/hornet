#include <iostream>
#include <sstream>
#include <string>
#include <memory>
#include <chrono>
#include <string.h>

#include "record.h"
#include "device.h"
#include "request.h"
#include "event.h"
#include "master.h"


using namespace std;

Device g_device;
RecordMap g_record_map;


// int main()
// {
//     cout << "start" << endl;

    // string meta_dir("/home/ghost/code/git-hub/hornet/data/");
    // string data_dir("/home/ghost/code/git-hub/hornet/data/");
    // Device d(meta_dir, data_dir, 0, 1024 * 1024 * 1024);

//     Device d(meta_dir, data_dir, 0, 1024 * 1024 * 1024);

//     cout << d.Init() << endl;
//     cout << d.Size() << endl;

//     auto start = chrono::system_clock::now();

//     char test_data[] = "123456789";
//     for (int i=0; i < 10 * 1024 * 1024; i++) {
//         Record r = {0};
//         r.id.data[0] = i;
//         r.dir.data[0] = i % 2;
//         r.length = 10;
//         d.Add(r, test_data);
//     }

// //   d.DumpMeta();
//     cout << d.Size()  << endl;
//     cout << chrono::duration_cast<chrono::milliseconds>(chrono::system_clock::now() - start).count() << endl;
// //   d.DumpMeta(); coast 2-3 secouds

//     start = chrono::system_clock::now();
//     d.DeleteByDir({0,0});
//     cout << d.Size() << endl;
//     cout << chrono::duration_cast<chrono::milliseconds>(chrono::system_clock::now() - start).count() << endl;
   
    
//     start = chrono::system_clock::now();    
//     d.DeleteByBlock(0);
// //   d.DumpMeta();    
//     cout << d.Size() << endl;
//     cout << chrono::duration_cast<chrono::milliseconds>(chrono::system_clock::now() - start).count() << endl;    

//     cout << "end" << endl;
    
//     return 0;
// }


int main()
{
    // string meta_dir("/home/ghost/code/git-hub/hornet/data/");
    // string data_dir("/home/ghost/code/git-hub/hornet/data/");
    // Device d(meta_dir, data_dir, 0, 1024 * 1024 * 1024);
    // d.Init();

    Master master("127.0.0.1", 8080, 1u);
    master.Forever();

    return 0;
}