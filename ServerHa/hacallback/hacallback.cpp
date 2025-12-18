#include "hacallback.h"
#include <stdio.h>
#include <ctime>

/*******************************************************************/
/*                  file:    	hacallback.h			           */
/*                  date:    	24/11/17                           */
/*                  author:  	still-her                           */
/*                  version: 	her-v1.00                          */
/*         		    name:       hacallback.so					   */
/*         		    说明:       生成库供ha模块回调  				 */
/*******************************************************************/

hasta stationstatus = init; //当前站点间状态
hasta Internastatus = init; //当前站点内状态

StaIntUpdateStateFun StaUpdateStateFun = NULL; 
StaIntUpdateStateFun IntUpdateStateFun = NULL; 


void nowtime2str(char *str){
    std::time_t now = std::time(0);
    std::tm* local_time = std::localtime(&now);
	
    int year = local_time->tm_year + 1900;
    int month = local_time->tm_mon + 1;
    int day = local_time->tm_mday;
    int hour = local_time->tm_hour;
    int minute = local_time->tm_min;
    int second = local_time->tm_sec;
	sprintf(str, "current time: %d-%d-%d %d:%d:%d", year, month, day, hour, minute, second);
	return ;
}


//   ha站点间发出告警回调函数
extern "C" void StationAlarmCodeCallback(int alert){
	char str[256] = {0};
	nowtime2str(str);
	printf((char*)"Err: <StationAlarmCodeCallback> %s alert(%d)", str, alert);
}

//   ha站点间发出告警回调函数
extern "C" void StationAlarmMsgCallback(const char* alert){
	char str[256] = {0};
	nowtime2str(str);
	printf((char*)"Err: <StationAlarmMsgCallback> %s alert(%s)", str, alert);
}

//   ha站点间变为主回调函数
extern "C" void StationMasterCallback(){

	char str[256] = {0};
	nowtime2str(str);
	printf((char*)"Err: <StationMasterCallback> %s Change status to Mstaer ", str);

	if (StaUpdateStateFun)
		StaUpdateStateFun(master);
	stationstatus = master;
}

//   ha站点间变为备回调函数
extern "C" void StationBackupCallback(){

	char str[256] = {0};
	nowtime2str(str);
	printf((char*)"Err: <StationBackupCallback> %s Change status to Backup ", str);
	
	if (StaUpdateStateFun)
		StaUpdateStateFun(backup);
	stationstatus = backup;
	
}


//   ha站点内发出告警回调函数
extern "C" void InternalAlarmCodeCallback(int alert){

	char str[256] = {0};
	nowtime2str(str);
	printf((char*)"Err: <InternalAlarmCodeCallback> %s alert(%d)", str, alert);
	
}

//   ha站点内发出告警回调函数
extern "C" void InternalAlarmMsgCallback(const char* alert){

	char str[256] = {0};
	nowtime2str(str);
	printf((char*)"Err: <InternalAlarmMsgCallback> %s alert(%s)", str, alert);
	
}

//   ha站点内变为主回调函数
extern "C" void InternalMasterCallback(){

	char str[256] = {0};
	nowtime2str(str);
	printf((char*)"Err: <InternalMasterCallback> %s  Change status to Mstaer ", str);
	
	if (IntUpdateStateFun)
		IntUpdateStateFun(master);
	Internastatus = master;
	
}

//   ha站点内变为备回调函数
extern "C" void InternalBackupCallback(){

	char str[256] = {0};
	nowtime2str(str);
	printf((char*)"Err: <InternalBackupCallback> %s  Change status to Backup ", str);
	
	if (IntUpdateStateFun)
		IntUpdateStateFun(backup);
	Internastatus = backup;
}

extern "C" void test(){
	printf("test\n");
}


