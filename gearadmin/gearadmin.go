package gearadmin

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// GearmanAdmin communicates with a gearman server.
type GearmanAdmin struct {
	conn io.ReadWriter
}

// NewGearmanAdmin takes in a connection and returns an object that interacts with gearman's admin protocol.
func NewGearmanAdmin(connection io.ReadWriter) GearmanAdmin {
	return GearmanAdmin{conn: connection}
}

// Status represents the status of a queue for a function as returned by the "status" command.
type Status struct {
	Function         string
	Total            int
	Running          int
	AvailableWorkers int
}

type PriorityStatus struct {
	Function         string
	HighQueued       int
	NormalQueued     int
	LowQueued        int
	AvailableWorkers int
}

// Worker represents a worker connected to gearman as returned by the "workers" command.
type Worker struct {
	Fd        string
	IPAddress string
	ClientID  string
	Functions []string
}

// Status returns the status of all function queues.
func (ga GearmanAdmin) Status() ([]Status, error) {
	var statuses []Status
	fmt.Fprintf(ga.conn, "status\n")
	scanner := bufio.NewScanner(ga.conn)
	for scanner.Scan() && scanner.Text() != "." {
		toks := strings.Split(scanner.Text(), "\t")
		if len(toks) != 4 {
			return statuses, fmt.Errorf("unexpected status: '%v'", scanner.Text())
		}
		total, err := strconv.Atoi(toks[1])
		if err != nil {
			return statuses, fmt.Errorf("could not parse total: '%v'", scanner.Text())
		}
		running, err := strconv.Atoi(toks[2])
		if err != nil {
			return statuses, fmt.Errorf("could not parse running: '%v'", scanner.Text())
		}
		available, err := strconv.Atoi(toks[3])
		if err != nil {
			return statuses, fmt.Errorf("could not parse available: '%v'", scanner.Text())
		}
		statuses = append(statuses, Status{
			Function:         toks[0],
			Total:            total,
			Running:          running,
			AvailableWorkers: available,
		})
	}
	return statuses, scanner.Err()
}

func (ga GearmanAdmin) PriorityStatus() ([]PriorityStatus, error) {
	var pStatuses []PriorityStatus
	fmt.Fprintf(ga.conn, "prioritystatus\n")
	scanner := bufio.NewScanner(ga.conn)

	for scanner.Scan() && scanner.Text() != "." {
		toks := strings.Split(scanner.Text(), "\t")

		if len(toks) != 5 {
			return pStatuses, fmt.Errorf("unexpected prioritystatus: '%v'", scanner.Text())
		}

		hq, err := strconv.Atoi(toks[1])
		if err != nil {
			return pStatuses, fmt.Errorf("could not parse HIGH-QUEUED: '%v'", scanner.Text())
		}
		nq, err := strconv.Atoi(toks[2])
		if err != nil {
			return pStatuses, fmt.Errorf("could not parse NORMAL-QUEUED: '%v'", scanner.Text())
		}
		lq, err := strconv.Atoi(toks[3])
		if err != nil {
			return pStatuses, fmt.Errorf("could not parse LOW-QUEUED: '%v'", scanner.Text())
		}
		available, err := strconv.Atoi(toks[4])
		if err != nil {
			return pStatuses, fmt.Errorf("could not parse available: '%v'", scanner.Text())
		}
		pStatuses = append(pStatuses, PriorityStatus{
			Function:         toks[0],
			HighQueued:       hq,
			NormalQueued:     nq,
			LowQueued:        lq,
			AvailableWorkers: available,
		})
	}
	return pStatuses, scanner.Err()
}

// Workers returns a summary of workers connected to gearman.
func (ga GearmanAdmin) Workers() ([]Worker, error) {
	var workers []Worker
	fmt.Fprintf(ga.conn, "workers\n")
	scanner := bufio.NewScanner(ga.conn)
	for scanner.Scan() && scanner.Text() != "." {
		toks := strings.Split(scanner.Text(), " ")
		if len(toks) < 4 {
			return workers, fmt.Errorf("unexpected worker: '%v'", scanner.Text())
		}
		workers = append(workers, Worker{
			Fd:        toks[0],
			IPAddress: toks[1],
			ClientID:  toks[2],
			Functions: toks[4:],
		})
	}
	return workers, scanner.Err()
}

func (ga GearmanAdmin) Cancel(handle string) (bool, error) {
	fmt.Fprintf(ga.conn, fmt.Sprintf("cancel-job %v\n", handle))
	scanner := bufio.NewScanner(ga.conn)
	if scanner.Scan() {
		resp := scanner.Text()
		if resp == "OK" {
			return true, nil
		}
		return false, errors.New(resp)
	}
	return false, scanner.Err()
}
