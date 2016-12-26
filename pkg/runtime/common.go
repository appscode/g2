package runtime

var argc = []int{
	/* UNUSED" */ 0,
	/* CAN_DO */ 1,
	/* CANT_DO */ 1,
	/* RESET_ABILITIES */ 0,
	/* PRE_SLEEP */ 0,
	/* UNUSED */ 0,
	/* NOOP */ 0,
	/* SUBMIT_JOB */ 3,
	/* JOB_CREATED */ 1,
	/* GRAB_JOB */ 0,
	/* NO_JOB */ 0,
	/* JOB_ASSIGN */ 3,
	/* WORK_STATUS */ 3,
	/* WORK_COMPLETE */ 2,
	/* WORK_FAIL */ 1,
	/* GET_STATUS */ 1, //different from libgearman.cc
	/* ECHO_REQ */ 1,
	/* ECHO_RES */ 1,
	/* SUBMIT_JOB_BG */ 3,
	/* ERROR */ 2,
	/* STATUS_RES */ 5,
	/* SUBMIT_JOB_HIGH */ 3,
	/* SET_CLIENT_ID */ 1,
	/* CAN_DO_TIMEOUT */ 2,
	/* ALL_YOURS */ 0,
	/* WORK_EXCEPTION */ 2,
	/* OPTION_REQ */ 1,
	/* OPTION_RES */ 1,
	/* WORK_DATA */ 2,
	/* WORK_WARNING */ 2,
	/* GRAB_JOB_UNIQ */ 0,
	/* JOB_ASSIGN_UNIQ */ 4,
	/* SUBMIT_JOB_HIGH_BG */ 3,
	/* SUBMIT_JOB_LOW */ 3,
	/* SUBMIT_JOB_LOW_BG */ 3,
	/* SUBMIT_JOB_SCHED */ 8,
	/* SUBMIT_JOB_EPOCH */ 4,
}

func (i PT) ArgCount() int {
	switch {
	case 1 <= i && i <= 36:
		return argc[i]
	default:
		return 0
	}
}

func NewBuffer(l int) (buf []byte) {
	// TODO add byte buffer pool
	buf = make([]byte, l)
	return
}
