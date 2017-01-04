// Package gearadmin provides simple bindings to the gearman admin protocol: http://gearman.org/protocol/.
//
//
// Usage
//
// Here's an example program that outputs the status of all worker queues in gearman:
//
//         package main
//
//         import (
//         	"fmt"
//         	"github.com/Clever/gearadmin"
//         	"net"
//         )
//
//         func main() {
//         	c, err := net.Dial("tcp", "localhost:4730")
//         	if err != nil {
//         		panic(err)
//         	}
//         	defer c.Close()
//         	admin := gearadmin.NewGearmanAdmin(c)
//         	status, _ := admin.Status()
//         	fmt.Printf("%#v\n", status)
//         }
package gearadmin
