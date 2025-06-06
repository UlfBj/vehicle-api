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
	"os"
//	"strconv"
//	"strings"
	"time"

	"VISS-Go/VapiViss"
//	"github.com/akamensky/argparse"
//	"github.com/covesa/vissr/utils"
//	"github.com/google/uuid"
)

func subscribeOutUnpack(subscribeOut VapiViss.SubscribeOutput) {
	fmt.Printf(`Subscribe call status=%d`+"\n", subscribeOut.Status)
	if subscribeOut.Status == VapiViss.SUCCESSFUL {
		for i := 0; i<len(subscribeOut.Data); i++ {
			fmt.Printf("Path=%s\n", subscribeOut.Data[i].Path)
			for j := 0; j<len(subscribeOut.Data[i].Dp); j++ {
				fmt.Printf("  Value=%s Ts=%s\n", subscribeOut.Data[i].Dp[j].Value, subscribeOut.Data[i].Dp[j].Timestamp)
			}
		}
	} else {
		fmt.Printf("Subscribe() call failed. Error reason = %s.\n", subscribeOut.Error.Reason)
	}
}

func main() {
	vehicleGuid1 := "pseudoVin1"
	vehicleGuid2 := "pseudoVin2"
	vehicleGuid3 := "pseudoVin3"

	out := VapiViss.Connect(vehicleGuid1, "")
	if out.Error != nil {
		fmt.Printf("Could not connect to vehicle id =%s. Error = %s.\n", vehicleGuid1, out.Error.Reason)
		os.Exit(-1)
	} else {
		fmt.Printf("Connected to vehicle id =%s\nSupported protocols= [", vehicleGuid1)
		for i := 0; i< len(out.Protocol); i++ {
			fmt.Printf("%s ", out.Protocol[i])
		}
		fmt.Printf("]\n")
	}
	vehicle1 := out.VehicleId
	va1 := vehicle1.InitAccess(vehicle1)

	out = VapiViss.Connect(vehicleGuid2, "")
	if out.Error != nil {
		fmt.Printf("Could not connect to vehicle id =%s. Error = %s.\n", vehicleGuid2, out.Error.Reason)
		os.Exit(-1)
	} else {
		fmt.Printf("Connected to vehicle id =%s\n", vehicleGuid2)
	}
	vehicle2 := out.VehicleId
	va2 := vehicle2.InitAccess(vehicle2)

	out = VapiViss.Connect(vehicleGuid3, "")
	if out.Error != nil {
		fmt.Printf("Could not connect to vehicle id =%s. Error = %s.\n", vehicleGuid3, out.Error.Reason)
		os.Exit(-1)
	} else {
		fmt.Printf("Connected to vehicle id =%s\n", vehicleGuid3)
	}
	vehicle3 := out.VehicleId
	va3 := vehicle3.InitAccess(vehicle3)

	path := "Vehicle.CurrentLocation"
	filter := `{"variant":"paths","parameter":["Latitude", "Longitude"]}`
	fmt.Printf(`Get(vehicle1, %s, %s, "")`+"\n", path, filter)
	getOut := va1.Signal.Get(vehicle1, path, filter, "")
	fmt.Printf(`Get call status=%d`+"\n", getOut.Status)
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
	subscribeOut := va1.Signal.Subscribe(vehicle1, path, filter, "", subscribeOutUnpack)
	subscribeOutUnpack(subscribeOut)

	seatIdOut := va1.Service.Seating.GetPropertiesSeatId(vehicle1)
	if seatIdOut.Status == VapiViss.SUCCESSFUL {
		seatIdList1 := seatIdOut.SeatIds
		fmt.Printf("SeatId properties(%s):\n", VapiViss.GetVehicleName(vehicle1))
		for i := 0; i<len(seatIdList1); i++ {
			fmt.Printf("%s:",seatIdList1[i].RowName)
			for j := 0; j<len(seatIdList1[i].ColumnName); j++ {
				fmt.Printf("%s ",seatIdList1[i].ColumnName[j])
			}
			fmt.Printf("\n")
		}
	}
	va2.Service.HVAC.Service1(vehicle2)
	va3.Service.HVAC.Service1(vehicle3)

	time.Sleep(15 * time.Second)  //wait to receive a few events...

	vehicle2.Disconnect(vehicle2)
	vehicle3.Disconnect(vehicle3)
	vehicle1.Disconnect(vehicle1)

	va1.Service.Seating.GetPropertiesSeatId(vehicle1)
}
