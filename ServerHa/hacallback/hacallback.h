#pragma once

/*******************************************************************/
/*                  file:    	hacallback.h			           */
/*                  date:    	24/11/17                           */
/*                  author:  	still-her                           */
/*                  version: 	her-v1.00                          */
/*         		    name:       hacallback.so					   */
/*         		    说明:         生成库供ha模块回调  					   */
/*******************************************************************/

#ifdef __cplusplus
extern "C" {
#endif

	//1:备 2:主 0:初始化
	enum hasta
	{
		init	= 0,
		backup,
		master,
	};

	typedef void(*StaIntUpdateStateFun)(int sta);

	//   ha站点间发出告警回调函数
	void StationAlarmCodeCallback(int alert);

	//   ha站点间发出告警回调函数
	void StationAlarmMsgCallback(const char* alert);

	//   ha站点间变为主回调函数
	void StationMasterCallback();

	//   ha站点间变为备回调函数
	void StationBackupCallback();


	//   ha站点内发出告警回调函数
	void InternalAlarmCodeCallback(int alert);

	//   ha站点内发出告警回调函数
	void InternalAlarmMsgCallback(const char* alert);

	//   ha站点内变为主回调函数
	void InternalMasterCallback();

	//   ha站点内变为备回调函数
	void InternalBackupCallback();


	void test();
	
#ifdef __cplusplus
}
#endif

