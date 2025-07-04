/**
* (C) 2025 Ford Motor Company
*
* All files and artifacts in the repository at https://github.com/ulfbj/Vehicle-Service-API
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"fmt"
//	"encoding/json"
//	"io"
//	"io/fs"
//	"net/http"
//	"os"
//	"strconv"
	"strings"
	"time"

	"VISS-Go/VapiViss"
//	"github.com/akamensky/argparse"
//	"github.com/covesa/vissr/utils"
//	"github.com/google/uuid"
)

func subscribeOutUnpack(subscribeOut VapiViss.SubscribeOutput) {
	showServiceStatus(subscribeOut.Status, subscribeOut.Error)
	if subscribeOut.Status == VapiViss.SUCCESSFUL {
		fmt.Printf(`ServiceId=%d`+"\n", subscribeOut.ServiceId)
		for i := 0; i<len(subscribeOut.Data); i++ {
			fmt.Printf("Path=%s\n", subscribeOut.Data[i].Path)
			for j := 0; j<len(subscribeOut.Data[i].Dp); j++ {
				fmt.Printf("  Value=%s Ts=%s\n", subscribeOut.Data[i].Dp[j].Value, subscribeOut.Data[i].Dp[j].Timestamp)
			}
		}
	}
}

func moveSeatOutUnpack(moveSeatOut VapiViss.MoveSeatOutput) {
	showServiceStatus(moveSeatOut.Status, moveSeatOut.Error)
	if moveSeatOut.Status != VapiViss.FAILED {
		fmt.Printf("Position=%d\n", moveSeatOut.Position)
	}
}

func showServiceStatus(status VapiViss.ProcedureStatus, err *VapiViss.ErrorData) {
	fmt.Printf("Call status=%d\n", status)
	if err != nil {
		fmt.Printf("Error code=%d\n", err.Code)
		fmt.Printf("Error reason=%s\n", err.Reason)
		fmt.Printf("Error status=%s\n", err.Description)
	}
}

func main() {
	vehicleGuid1 := "pseudoVin1"

	var protocol string
	initOut := VapiViss.GetVehicle(vehicleGuid1)
	fmt.Printf("Initiated connection to vehicle id =%s\nSupported protocols= [", vehicleGuid1)
	for i := 0; i< len(initOut.Protocol); i++ {
		fmt.Printf("%s ", initOut.Protocol[i])
	}
	fmt.Printf("]\n")
	for i := 0; i < len(initOut.Protocol); i++ {
		if strings.Contains(initOut.Protocol[i], "ws") {
			protocol = initOut.Protocol[i]
		}
	}
	fmt.Printf("protocol =%s\n", protocol)
	vehicle1 := initOut.VehicleId
	out := VapiViss.Connect(vehicle1, protocol, "")
	if out.Error != nil {
		fmt.Printf("Could not connect to vehicle id =%s. Error = %s.\n", vehicleGuid1, out.Error.Reason)
	} else {
		fmt.Printf("Connected to vehicle id =%s\n", vehicleGuid1)
	}

	path := "Vehicle.CurrentLocation"
	filter := `{"variant":"paths","parameter":["Latitude", "Longitude"]}`
	VapiViss.SelectProtocol(vehicle1, protocol)
	fmt.Printf(`Get(vehicle1, %s, %s, "")`+"\n", path, filter)
	getOut := VapiViss.Get(vehicle1, path, filter, "")
	showServiceStatus(getOut.Status,getOut.Error)
	if getOut.Status == VapiViss.SUCCESSFUL {
		for i := 0; i<len(getOut.Data); i++ {
			fmt.Printf("Path=%s\n", getOut.Data[i].Path)
			for j := 0; j<len(getOut.Data[i].Dp); j++ {
				fmt.Printf("  Value=%s Ts=%s\n", getOut.Data[i].Dp[j].Value, getOut.Data[i].Dp[j].Timestamp)
			}
		}
	} else {
		fmt.Printf("Get() call to vehicle id =%s failed. Error reason = %s.\n", vehicleGuid1, getOut.Error.Reason)
	}

	filter = `[{"variant":"paths","parameter":["Latitude", "Longitude"]}, {"variant":"timebased","parameter":{"period":"1000"}}]`
	fmt.Printf(`Subscribe(vehicle1, %s, %s, "", subscribeOutUnpack)`+"\n", path, filter)
	subscribeOut := VapiViss.Subscribe(vehicle1, path, filter, "", subscribeOutUnpack)
	subscribeOutUnpack(subscribeOut)

	fmt.Printf("Sleep for 3 secs to lreceive a few events...\n")
	time.Sleep(3000 * time.Millisecond)

	fmt.Printf("Unsubscribe(vehicle1, %d)\n", subscribeOut.ServiceId)
	unsubscribeOut := VapiViss.Unsubscribe(vehicle1, subscribeOut.ServiceId)
	showServiceStatus(unsubscribeOut.Status,unsubscribeOut.Error)

	fmt.Printf("GetPropertiesSeating(vehicle1)\n")
	getPropertiesSeatingOut := VapiViss.GetPropertiesSeating(vehicle1)
	showServiceStatus(getPropertiesSeatingOut.Status, getPropertiesSeatingOut.Error)
	if getPropertiesSeatingOut.Status == VapiViss.SUCCESSFUL {
		seatIdList := getPropertiesSeatingOut.Id
		fmt.Printf("Seating properties:\n")
		fmt.Printf("Seat Ids:\n")
		for i := 0; i<len(seatIdList); i++ {
			fmt.Printf("%s:",seatIdList[i].RowName)
			for j := 0; j<len(seatIdList[i].ColumnName); j++ {
				fmt.Printf("%s ",seatIdList[i].ColumnName[j])
			}
			fmt.Printf("\n")
		}
		fmt.Printf("Seat movement types:\n")
		for i := 0; i<len(getPropertiesSeatingOut.Movement); i++ {
			fmt.Printf("%s:%s\n",getPropertiesSeatingOut.Movement[i].Name, getPropertiesSeatingOut.Movement[i].Description)
		}
	}

	// to simulate an execution duration for MoveSeat, set it to an initial value different from what MoveSeat invokes
	longitudinalPath := "Vehicle.Cabin.Seat.Row1.DriverSide.Position"
	fmt.Printf(`Set(vehicle1, %s, 2, "")`+"\n", longitudinalPath)
	setOut := VapiViss.Set(vehicle1, longitudinalPath, "2", "")
	showServiceStatus(setOut.Status,setOut.Error)

	fmt.Printf("Sleep for 20 secs to let execution duration from Set finish..\n")
	time.Sleep(20 * time.Second)  //wait to let execution duration from Set finish 

	var seatId VapiViss.MatrixId
	seatId.RowName = getPropertiesSeatingOut.Id[0].RowName
	seatId.ColumnName = getPropertiesSeatingOut.Id[0].ColumnName[0]
	fmt.Printf("MoveSeat(vehicle1, seatId, longitudinal, %D, moveSeatOutUnpack)\n", VapiViss.BACKWARD)
	moveSeatOut := VapiViss.MoveSeat(vehicle1, seatId, "longitudinal", VapiViss.BACKWARD, "", moveSeatOutUnpack)
	showServiceStatus(moveSeatOut.Status, moveSeatOut.Error)

	time.Sleep(30 * time.Second)  //wait to let execution duration from MoveSeat finish 

/*	fmt.Printf(`Get(vehicle1, %s, %s, "")`+"\n", longitudinalPath, filter)
	for i := 0; i < 10; i++ {
		getOut = VapiViss.Get(vehicle1, longitudinalPath, filter, "")
		showServiceStatus(getOut.Status,getOut.Error)
		if getOut.Status == VapiViss.SUCCESSFUL {
			for i := 0; i<len(getOut.Data); i++ {
				fmt.Printf("Path=%s\n", getOut.Data[i].Path)
				for j := 0; j<len(getOut.Data[i].Dp); j++ {
					fmt.Printf("  Value=%s Ts=%s\n", getOut.Data[i].Dp[j].Value, getOut.Data[i].Dp[j].Timestamp)
				}
			}
		} else {
			fmt.Printf("Get() call to vehicle id =%s failed. Error reason = %s.\n", vehicleGuid1, getOut.Error.Reason)
		}
		time.Sleep(1 * time.Second)
	}*/

//	time.Sleep(20 * time.Second)  //wait to check if unsub leads to ws tear down

/*	path = "Vehicle.Cabin.Seat.Row1.DriverSide.Position"
	VapiViss.SelectProtocol(vehicle2, protocol2)
	fmt.Printf(`Get(vehicle2, %s, %s, "")`+"\n", path, filter)
	getOut = VapiViss.Get(vehicle2, path, filter, "")
	showServiceStatus(getOut.Status,getOut.Error)
	if getOut.Status == VapiViss.SUCCESSFUL {
		for i := 0; i<len(getOut.Data); i++ {
			fmt.Printf("Path=%s\n", getOut.Data[i].Path)
			for j := 0; j<len(getOut.Data[i].Dp); j++ {
				fmt.Printf("  Value=%s Ts=%s\n", getOut.Data[i].Dp[j].Value, getOut.Data[i].Dp[j].Timestamp)
			}
		}
	} else {
		fmt.Printf("Get() call to vehicle id =%s failed. Error reason = %s.\n", vehicleGuid1, getOut.Error.Reason)
	}

	fmt.Printf("Disconnect(vehicle2, %s)\n", protocol2)
	VapiViss.Disconnect(vehicle2, protocol2)
//	VapiViss.Disconnect(vehicle3, protocol3)*/
	fmt.Printf("Disconnect(vehicle1, %s)\n", protocol)
	disconnectOut := VapiViss.Disconnect(vehicle1, protocol)
	showServiceStatus(disconnectOut.Status, disconnectOut.Error)


	fmt.Printf("GetPropertiesSeatId(vehicle1)\n")
	getPropertiesSeatingOut = VapiViss.GetPropertiesSeating(vehicle1)
	showServiceStatus(getPropertiesSeatingOut.Status, getPropertiesSeatingOut.Error)

	releaseOut := VapiViss.ReleaseVehicle(vehicle1)
	showServiceStatus(releaseOut.Status, releaseOut.Error)
}
