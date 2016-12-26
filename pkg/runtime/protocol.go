package runtime

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
	queueSize = 8
	// read buffer size
	bufferSize = 1024
	// min packet length
	minPacketLength = 12

	// \x00REQ
	Req    = 5391697
	ReqStr = "\x00REQ"
	// \x00RES
	Res    = 5391699
	ResStr = "\x00RES"

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

const (
	CAN_DO          = iota + 1 //   1            REQ    Worker
	CANT_DO                    //   REQ    Worker
	RESET_ABILITIES            //   REQ    Worker
	PRE_SLEEP                  //   REQ    Worker
	UNUSED                     //   -      -
	NOOP                       //   RES    Worker
	SUBMIT_JOB                 //   REQ    Client
	JOB_CREATED                //   RES    Client
	GRAB_JOB                   //   REQ    Worker
	NO_JOB                     //   RES    Worker
	JOB_ASSIGN                 //   RES    Worker
	WORK_STATUS                //   REQ    Worker

	//     RES    Client
	WORK_COMPLETE //   REQ    Worker

	//    RES    Client
	WORK_FAIL //  REQ    Worker

	//    RES    Client
	GET_STATUS //   REQ    Client

	ECHO_REQ        //  REQ    Client/Worker
	ECHO_RES        //  RES    Client/Worker
	SUBMIT_JOB_BG   //   REQ    Client
	ERROR           //   RES    Client/Worker
	STATUS_RES      //   RES    Client
	SUBMIT_JOB_HIGH //   REQ    Client
	SET_CLIENT_ID   //  REQ    Worker
	CAN_DO_TIMEOUT  //   REQ    Worker
	ALL_YOURS       //   REQ    Worker
	WORK_EXCEPTION  //   REQ    Worker

	//     RES    Client
	OPTION_REQ //   REQ    Client/Worker

	OPTION_RES //   RES    Client/Worker
	WORK_DATA  //   REQ    Worker

	//    RES    Client
	WORK_WARNING //  REQ    Worker

	//    RES    Client
	GRAB_JOB_UNIQ //   REQ    Worker

	JOB_ASSIGN_UNIQ    //   RES    Worker
	SUBMIT_JOB_HIGH_BG //  REQ    Client
	SUBMIT_JOB_LOW     //  REQ    Client
	SUBMIT_JOB_LOW_BG  //  REQ    Client
	SUBMIT_JOB_SCHED   //  REQ    Client
	SUBMIT_JOB_EPOCH   //   36 REQ    Client
)
