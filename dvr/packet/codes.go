package packet

type Code uint16

const (
	// LOGIN_REQ Code = 999
	LOGIN_REQ       Code = 1000
	LOGIN_RSP       Code = 1001
	LOGOUT_REQ      Code = 1001
	LOGOUT_RSP      Code = 1002
	FORCELOGOUT_REQ Code = 1003
	FORCELOGOUT_RSP Code = 1004
	KEEPALIVE_REQ   Code = 1005
	KEEPALIVE_RSP   Code = 1006

	MONITOR_REQ       Code = 1410
	MONITOR_RSP       Code = 1411
	MONITOR_DATA      Code = 1412
	MONITOR_CLAIM     Code = 1413
	MONITOR_CLAIM_RSP Code = 1414

	SYSMANAGER_REQ Code = 1450
	SYSMANAGER_RSP Code = 1451
)

// const (
//     Login            Code = 1000
//     KeepAlive        Code = 1006
//     SystemInfo       Code = 1020
//     NetWorkNetCommon Code = 1042
//     General          Code = 1042
//     ChannelTitle     Code = 1046
//     SystemFunction   Code = 1360
//     EnCapability     Code = 1360
//     OPPTZControl     Code = 1400
//     MONITOR_DATA     Code = 1412
//     OPMonitor        Code = 1413
//     OPTalk           Code = 1434
//     OPTimeSetting    Code = 1450
//     OPMachine        Code = 1450
//     OPTimeQuery      Code = 1452
//     AuthorityList    Code = 1470
//     Users            Code = 1472
//     Groups           Code = 1474
//     AddGroup         Code = 1476
//     ModifyGroup      Code = 1478
//     DelGroup         Code = 1480
//     AddUser          Code = 1482
//     ModifyUser       Code = 1484
//     DelUser          Code = 1486
//     ModifyPassword   Code = 1488
//     AlarmSet         Code = 1500
//     OPNetAlarm       Code = 1506
//     AlarmInfo        Code = 1504
//     OPSendFile       Code = 1522
//     OPSystemUpgrade  Code = 1525
//     OPNetKeyboard    Code = 1550
//     OPSNAP           Code = 1560
//     OPMailTest       Code = 1636
// )

var Commands = map[Code]string{
	MONITOR_CLAIM:  "OPMonitor",
	SYSMANAGER_REQ: "OPTimeSetting",
}
