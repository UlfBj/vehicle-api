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
		fmt.Printf("Position=%f\n", moveSeatOut.Position)
	}
}

func seatConfigOutUnpack(configureSeatOut VapiViss.ConfigureSeatOutput) {
	fmt.Printf("ConfigureSeat:")
	showServiceStatus(configureSeatOut.Status, configureSeatOut.Error)
	for i := 0; i < len(configureSeatOut.Configured); i++ {
		fmt.Printf("Configured:%s, Position:%f\n", configureSeatOut.Configured[i].MovementType, configureSeatOut.Configured[i].Position)
	}
}

func massageOutUnpack(massageOut VapiViss.MassageOutput) {
	fmt.Printf("\nMassage execution status:\n")
	showServiceStatus(massageOut.Status, massageOut.Error)
}

func showServiceStatus(status VapiViss.ProcedureStatus, err *VapiViss.ErrorData) {
	fmt.Printf("Call status=%s\n", translateStatus(status))
	if err != nil {
		fmt.Printf("Error code=%d\n", err.Code)
		fmt.Printf("Error reason=%s\n", err.Reason)
		fmt.Printf("Error status=%s\n", err.Description)
	}
}

func translateStatus(status VapiViss.ProcedureStatus) string {
	switch status {
		case VapiViss.SUCCESSFUL: return "SUCCESSFUL"
		case VapiViss.ONGOING: return "ONGOING"
		case VapiViss.FAILED: return "FAILED"
	}
	return "UNKNOWN"
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
	if out.Status != VapiViss.SUCCESSFUL {
		fmt.Printf("Could not connect to vehicle id =%s. Error = %s.\n", vehicleGuid1, out.Error.Reason)
		return
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

	fmt.Printf("Sleep for 3 secs to receive a few events...\n")
	time.Sleep(3000 * time.Millisecond)

	fmt.Printf("Unsubscribe(vehicle1, %d)\n", subscribeOut.ServiceId)
	unsubscribeOut := VapiViss.Unsubscribe(vehicle1, subscribeOut.ServiceId)
	showServiceStatus(unsubscribeOut.Status,unsubscribeOut.Error)

	fmt.Printf("GetPropertiesSeating(vehicle1)\n")
	getPropertiesSeatingOut := VapiViss.GetPropertiesSeating(vehicle1)
	showServiceStatus(getPropertiesSeatingOut.Status, getPropertiesSeatingOut.Error)
	if getPropertiesSeatingOut.Status == VapiViss.SUCCESSFUL {
		fmt.Printf("Seating properties:\n")
		for i := 0; i<len(getPropertiesSeatingOut.Properties); i++ {
			for j := 0; j<len(getPropertiesSeatingOut.Properties[i].Column); j++ {
				fmt.Printf("Seat Id= %s, %s\n", getPropertiesSeatingOut.Properties[i].RowName, getPropertiesSeatingOut.Properties[i].Column[j].Name)
				fmt.Printf("Movement support: ")
				for k := 0; k<len(getPropertiesSeatingOut.Properties[i].Column[j].MovementSupport); k++ {
					fmt.Printf("%s, ",getPropertiesSeatingOut.Properties[i].Column[j].MovementSupport[k])
				}
				fmt.Printf("\n")
				fmt.Printf("Massage support: ")
				for k := 0; k<len(getPropertiesSeatingOut.Properties[i].Column[j].MassageSupport); k++ {
					fmt.Printf("%s, ",getPropertiesSeatingOut.Properties[i].Column[j].MassageSupport[k])
				}
				fmt.Printf("\n")
			}
		}
	}

	// to simulate an execution duration for MoveSeat, set it to an initial value different from what MoveSeat invokes
	longitudinalPath := "Vehicle.Cabin.Seat.Row1.DriverSide.Position"
	fmt.Printf(`Set(vehicle1, %s, 2, "")`+"\n", longitudinalPath)
	setOut := VapiViss.Set(vehicle1, longitudinalPath, "2", "")
	showServiceStatus(setOut.Status,setOut.Error)

	fmt.Printf("Sleep for 12 secs to let execution duration from Set finish..\n")
	time.Sleep(12 * time.Second)

	var seatId VapiViss.MatrixId
	seatId.RowName = getPropertiesSeatingOut.Properties[0].RowName
	seatId.ColumnName = getPropertiesSeatingOut.Properties[0].Column[0].Name
	fmt.Printf("MoveSeat(vehicle1, seatId, LONGITUDINAL, %d, moveSeatOutUnpack)\n", VapiViss.BACKWARD)
	moveSeatOut := VapiViss.MoveSeat(vehicle1, seatId, VapiViss.LONGITUDINAL, VapiViss.BACKWARD, "", moveSeatOutUnpack)
	showServiceStatus(moveSeatOut.Status, moveSeatOut.Error)

	fmt.Printf("Sleep for 5 secs to let execution duration from moveSeat get about half way..\n")
	time.Sleep(5 * time.Second)

	fmt.Printf(`CancelService(vehicle1, %d)`+"\n", moveSeatOut.ServiceId)
	cancelServiceOut := VapiViss.CancelService(vehicle1, moveSeatOut.ServiceId)
	showServiceStatus(cancelServiceOut.Status,cancelServiceOut.Error)

	seatConfig := make([]VapiViss.SeatConfig, 2)
	seatConfig[0].MovementType = VapiViss.LONGITUDINAL
	seatConfig[0].Position = 10
	seatConfig[1].MovementType = VapiViss.LUMBAR
	seatConfig[1].Position = 50
	fmt.Printf("ConfigureSeat(vehicle1, seatId, seatConfig, '', seatConfigOutUnpack)\n")
	configSeatOut := VapiViss.ConfigureSeat(vehicle1, seatId, seatConfig, "", seatConfigOutUnpack)
	seatConfigOutUnpack(configSeatOut)

	fmt.Printf("Sleep for 12 secs to let seat configuration execution to finish\n")
	time.Sleep(12 * time.Second)

	fmt.Printf("ActivateMassage(vehicle1, seatId, ROLL, 50, 5, '', massageOutUnpack)\n")
	massageOut := VapiViss.ActivateMassage(vehicle1, seatId, VapiViss.ROLL, 50, 5, "", massageOutUnpack)
	massageOutUnpack(massageOut)

	fmt.Printf("Sleep for 7 secs to let massage execution duration=5s to finish\n")
	time.Sleep(7 * time.Second)

	fmt.Printf("Disconnect(vehicle1, %s)\n", protocol)
	disconnectOut := VapiViss.Disconnect(vehicle1, protocol)
	showServiceStatus(disconnectOut.Status, disconnectOut.Error)


	fmt.Printf("GetPropertiesSeating(vehicle1)\n")
	getPropertiesSeatingOut = VapiViss.GetPropertiesSeating(vehicle1)
	showServiceStatus(getPropertiesSeatingOut.Status, getPropertiesSeatingOut.Error)

	releaseOut := VapiViss.ReleaseVehicle(vehicle1)
	showServiceStatus(releaseOut.Status, releaseOut.Error)
}
