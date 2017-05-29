package runtime

import (
	"fmt"
)

/*
Binary Packet
-------------

Requests and responses are encapsulated by a binary packet. A binary
packet consists of a header which is optionally followed by data. The
header is:

4 byte magic code - This is either "\0REQ" for requests or "\0RES"
                    for responses.

4 byte type       - A big-endian (network-order) integer containing
                    an enumerated packet type. Possible values are:

                    #   Name                Magic  Type
                    1   CAN_DO              REQ    Worker
                    2   CANT_DO             REQ    Worker
                    3   RESET_ABILITIES     REQ    Worker
                    4   PRE_SLEEP           REQ    Worker
                    5   (unused)            -      -
                    6   NOOP                RES    Worker
                    7   SUBMIT_JOB          REQ    Client
                    8   JOB_CREATED         RES    Client
                    9   GRAB_JOB            REQ    Worker
                    10  NO_JOB              RES    Worker
                    11  JOB_ASSIGN          RES    Worker
                    12  WORK_STATUS         REQ    Worker
                                            RES    Client
                    13  WORK_COMPLETE       REQ    Worker
                                            RES    Client
                    14  WORK_FAIL           REQ    Worker
                                            RES    Client
                    15  GET_STATUS          REQ    Client
                    16  ECHO_REQ            REQ    Client/Worker
                    17  ECHO_RES            RES    Client/Worker
                    18  SUBMIT_JOB_BG       REQ    Client
                    19  ERROR               RES    Client/Worker
                    20  STATUS_RES          RES    Client
                    21  SUBMIT_JOB_HIGH     REQ    Client
                    22  SET_CLIENT_ID       REQ    Worker
                    23  CAN_DO_TIMEOUT      REQ    Worker
                    24  ALL_YOURS           REQ    Worker
                    25  WORK_EXCEPTION      REQ    Worker
                                            RES    Client
                    26  OPTION_REQ          REQ    Client/Worker
                    27  OPTION_RES          RES    Client/Worker
                    28  WORK_DATA           REQ    Worker
                                            RES    Client
                    29  WORK_WARNING        REQ    Worker
                                            RES    Client
                    30  GRAB_JOB_UNIQ       REQ    Worker
                    31  JOB_ASSIGN_UNIQ     RES    Worker
                    32  SUBMIT_JOB_HIGH_BG  REQ    Client
                    33  SUBMIT_JOB_LOW      REQ    Client
                    34  SUBMIT_JOB_LOW_BG   REQ    Client
                    35  SUBMIT_JOB_SCHED    REQ    Client
                    36  SUBMIT_JOB_EPOCH    REQ    Client

4 byte size       - A big-endian (network-order) integer containing
                    the size of the data being sent after the header.

Arguments given in the data part are separated by a NULL byte, and
the last argument is determined by the size of data after the last
NULL byte separator. All job handle arguments must not be longer than
64 bytes, including NULL terminator.
*/

const (
	Network = "tcp"
	// queue size
	QueueSize = 8
	// read buffer size
	BufferSize = 4096
	// min packet length
	MinPacketLength = 12

	// \x00REQ
	Req    = 5391697
	ReqStr = "\x00REQ"
	// \x00RES
	Res            = 5391699
	ResStr         = "\x00RES"
	DefaultTimeout = 20 // 20 Seconds

	HANDLE_SHAKE_HEADER_LENGTH = 12
)

const (
	// Job type
	JobNormal = iota
	// low level
	JobLow
	// high level
	JobHigh
)

type PT uint32

const (
	PT_CanDo          PT = iota + 1 //   1            REQ    Worker
	PT_CantDo                       //   REQ    Worker
	PT_ResetAbilities               //   REQ    Worker
	PT_PreSleep                     //   REQ    Worker
	_
	PT_Noop       //   RES    Worker
	PT_SubmitJob  //   REQ    Client
	PT_JobCreated //   RES    Client
	PT_GrabJob    //   REQ    Worker
	PT_NoJob      //   RES    Worker
	PT_JobAssign  //   RES    Worker
	PT_WorkStatus //   REQ    Worker

	//     RES    Client
	PT_WorkComplete //   REQ    Worker

	//    RES    Client
	PT_WorkFail //  REQ    Worker

	//    RES    Client
	PT_GetStatus //   REQ    Client

	PT_EchoReq       //  REQ    Client/Worker
	PT_EchoRes       //  RES    Client/Worker
	PT_SubmitJobBG   //   REQ    Client
	PT_Error         //   RES    Client/Worker
	PT_StatusRes     //   RES    Client
	PT_SubmitJobHigh //   REQ    Client
	PT_SetClientId   //  REQ    Worker
	PT_CanDoTimeout  //   REQ    Worker
	PT_AllYours      //   REQ    Worker
	PT_WorkException //   REQ    Worker

	//     RES    Client
	PT_OptionReq //   REQ    Client/Worker

	PT_OptionRes //   RES    Client/Worker
	PT_WorkData  //   REQ    Worker

	//    RES    Client
	PT_WorkWarning //  REQ    Worker

	//    RES    Client
	PT_GrabJobUniq //   REQ    Worker

	PT_JobAssignUniq   //   RES    Worker
	PT_SubmitJobHighBG //  REQ    Client
	PT_SubmitJobLow    //  REQ    Client
	PT_SubmitJobLowBG  //  REQ    Client
	PT_SubmitJobSched  //  REQ    Client
	PT_SubmitJobEpoch  //   36 REQ    Client

	// New Codes (ref: https://github.com/gearman/gearmand/commit/eabf8a01030a16c80bada1e06c4162f8d129a5e8)
	PT_SubmitReduceJob           // REQ    Client
	PT_SubmitReduceJobBackground // REQ    Client
	PT_GrabJobAll                // REQ    Worker
	PT_JobAssignAll              // RES    Worker
	PT_GetStatusUnique           // REQ    Client
	PT_StatusResUnique           // RES    Client
)

func (i PT) Int() int {
	return int(i)
}

func (i PT) Uint32() uint32 {
	return uint32(i)
}

func NewPT(cmd uint32) (PT, error) {
	if cmd >= PT_CanDo.Uint32() && cmd <= PT_SubmitJobEpoch.Uint32() {
		return PT(cmd), nil
	}
	if cmd >= PT_SubmitReduceJob.Uint32() && cmd <= PT_StatusResUnique.Uint32() {
		return PT(cmd), fmt.Errorf("Unsupported packet type %v", cmd)
	}
	return PT(cmd), fmt.Errorf("Invalid packet type %v", cmd)
}
