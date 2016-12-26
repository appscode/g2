package runtime

import (
	"strconv"
)

func CmdDescription(cmd uint32) string {
	if int(cmd) >= len(cmdTable) {
		return "unknown command " + strconv.Itoa(int(cmd))
	}

	return cmdTable[cmd].str
}

func ArgCount(cmd uint32) int {
	return cmdTable[cmd].argc
}

type cmdinfo struct {
	cmd  uint32
	str  string
	argc int
}

var cmdTable = []cmdinfo{
	{0, "UNUSED", 0},
	{1, "CAN_DO", 1},
	{2, "CANT_DO", 1},
	{3, "RESET_ABILITIES", 0},
	{4, "PRE_SLEEP", 0},
	{5, "UNUSED", 0},
	{6, "NOOP", 0},
	{7, "SUBMIT_JOB", 3},
	{8, "JOB_CREATED", 1},
	{9, "GRAB_JOB", 0},
	{10, "NO_JOB", 0},
	{11, "JOB_ASSIGN", 3},
	{12, "WORK_STATUS", 3},

	{13, "WORK_COMPLETE", 2},

	{14, "WORK_FAIL", 1},

	{15, "GET_STATUS", 1}, //different from libgearman.cc
	{16, "ECHO_REQ", 1},
	{17, "ECHO_RES", 1},
	{18, "SUBMIT_JOB_BG", 3},
	{19, "ERROR", 2},
	{20, "STATUS_RES", 5},
	{21, "SUBMIT_JOB_HIGH", 3},
	{22, "SET_CLIENT_ID", 1},
	{23, "CAN_DO_TIMEOUT", 2},
	{24, "ALL_YOURS", 0},
	{25, "WORK_EXCEPTION", 2},

	{26, "OPTION_REQ", 1},
	{27, "OPTION_RES", 1},
	{28, "WORK_DATA", 2},

	{29, "WORK_WARNING", 2},

	{30, "GRAB_JOB_UNIQ", 0},
	{31, "JOB_ASSIGN_UNIQ", 4},
	{32, "SUBMIT_JOB_HIGH_BG", 3},
	{33, "SUBMIT_JOB_LOW", 3},
	{34, "SUBMIT_JOB_LOW_BG", 3},
	{35, "SUBMIT_JOB_SCHED", 8},
	{36, "SUBMIT_JOB_EPOCH", 4},
}
