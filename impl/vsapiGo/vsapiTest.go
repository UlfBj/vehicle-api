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
//	"time"

	"vsapiTest/vsapiViss"
//	"github.com/akamensky/argparse"
//	"github.com/covesa/vissr/utils"
//	"github.com/google/uuid"
)

func main() {
	vehicleId := "pseudoVin1"
	vehicleId2 := "pseudoVin2"
	vehicleId3 := "pseudoVin3"
	vehicleServices := vsapiViss.VehicleConnect(vehicleId)
	if vehicleServices == nil {
		fmt.Printf("Could not connect to vehicle id =%s. Exiting.\n", vehicleId)
		os.Exit(-1)
	} else {
		fmt.Printf("Connected to vehicle id =%s\n", vehicleId)
	}
	vehicle2Services := vsapiViss.VehicleConnect(vehicleId2)
	if vehicle2Services == nil {
		fmt.Printf("Could not connect to vehicle id =%s. Exiting.\n", vehicleId2)
		os.Exit(-1)
	} else {
		fmt.Printf("Connected to vehicle id =%s\n", vehicleId2)
	}
	vehicle3Services := vsapiViss.VehicleConnect(vehicleId3)
	if vehicle3Services == nil {
		fmt.Printf("Could not connect to vehicle id =%s. Exiting.\n", vehicleId3)
		os.Exit(-1)
	} else {
		fmt.Printf("Connected to vehicle id =%s\n", vehicleId3)
	}
	vehicleServices.Seat.Service1(vehicleServices.VehicleId, "forward")
	vehicleServices.HVAC.Service1(vehicleServices.VehicleId)

	vehicle2Services.Disconnect(vehicle2Services.VehicleId)
	vehicle3Services.Disconnect(vehicle3Services.VehicleId)
	vehicleServices.Disconnect(vehicleServices.VehicleId)

	vehicleServices.Seat.Service1(vehicleServices.VehicleId, "backward")
}

