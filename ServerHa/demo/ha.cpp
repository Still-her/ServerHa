#include "libha.h"
#include "unistd.h"


//export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:../
//g++ ha.cpp ../hacallback/hacallback.cpp -o ha -I ../hacallback/ -I ../ -L../ -lha
//g++ ha.cpp ../hacallback/hacallback.cpp -o ha -I ../hacallback/ -I ../ -L../ -lha -lpthread -lresolv

int main(){
    Handlesignal();
    HaRuntime();
	HaLoginit();
	HaInternal();
	HaStaInitRun();
	HaStaNegotitae();
	HaStaStationPing();
	HaRouter();

	while(true){
		sleep(3);
	}
}