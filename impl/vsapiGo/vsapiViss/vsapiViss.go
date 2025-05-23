/**
* (C) 2025 Ford Motor Company
*
* All files and artifacts in the repository at https://github.com/ulfbj/Vehicle-Service-API
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package vsapiViss

import (
	"fmt"
//	"encoding/json"
//	"io"
//	"io/fs"
//	"net/http"
//	"os"
//	"strconv"
//	"strings"
//	"time"

//	"github.com/akamensky/argparse"
//	"github.com/covesa/vissr/utils"
//	"github.com/google/uuid"
)

type Vehicle struct {
	VehicleId string
	Seat SeatServices
	HVAC HVACServices
}

type SeatServices struct {
	Service1 func(string, string)
	seatMove func(string, DirectionType, uint8, func(MoveOutput)) MoveOutput
}

type HVACServices struct {
	Service1 func(string)
}

type vehicleInstance struct {
	vehicle Vehicle
	nextInstance *vehicleInstance
	vehicleSocket string
}

var vehicleList *vehicleInstance

func VehicleConnect(vehicleId string) *Vehicle {
	var instance vehicleInstance
	instance.vehicle.VehicleId = vehicleId
	//TODO:call vehicle server with service inquiry, update service function pointers
	instance.vehicle.Seat.Service1 = seatService1
	instance.vehicle.HVAC.Service1 = hvacService1
	addVehicle(&instance)
	return &instance.vehicle
}

func (v *Vehicle) Disconnect(vehicleId string) {
	removeVehicle(vehicleId)
}

func addVehicle(instance *vehicleInstance) {
	if vehicleList == nil {
		vehicleList = instance
	} else {
		iterator := vehicleList
		for iterator.nextInstance != nil {
			iterator = iterator.nextInstance
		}
		iterator.nextInstance = instance
	}
}

func removeVehicle(vehicleId string) {
	if vehicleList == nil {
		return
	} else {
		iterator := &vehicleList
		for *iterator != nil {
			if (*iterator).vehicle.VehicleId == vehicleId {
				*iterator =(*iterator).nextInstance
				fmt.Printf("Disconnected to vehicle id=%s()\n", vehicleId)
				break
			}
			iterator = &(*iterator).nextInstance
		}
	}
}

func isConnected(vehicleId string) bool {
	if vehicleList == nil {
		return false
	} else {
		iterator := vehicleList
		for iterator != nil {
			if iterator.vehicle.VehicleId == vehicleId {
				return true
			}
			iterator = iterator.nextInstance
		}
	}
	return false
}

func VehicleStatus(vehicleId string) {
	if isConnected(vehicleId) {
		fmt.Printf("Vehicle is connected\n")
	} else {
		fmt.Printf("Vehicle is not connected\n")
	}
}

func seatService1(vehicleId string, move string) {
	if isConnected(vehicleId) {
		fmt.Printf("seatService1(): move =%s\n", move)
	} else {
		fmt.Printf("seatService1(): Vehicle is not connected\n")
	}
}

func hvacService1(vehicleId string) {
	if isConnected(vehicleId) {
		fmt.Printf("hvacService1()\n")
	} else {
		fmt.Printf("hvacService1(): Vehicle is not connected\n")
	}
}

type ServiceStatus uint8
const (
	NOT_STARTED = 2
	ONGOING = 1
	SUCCESS = 0
	FAILED = -1
)

//Seat Services:
type DirectionType uint8
const (
	Longitudal = iota  //Forward-backward direction of the vehicle; 0=forward-most, 100=backward-most
	Lateral            // Left-right direction of the vehicle; 0=left-most, 100=right-most
	Vertical           // Up-down direction of the vehicle; 0=down-most, 100=up-most
	Tilt               // Seat forward-backward tilt; 0=forward-tilt-most, 100=backward-tilt-most
	Lumbar             // Seat increase-decrease lumbar; 0=decrease-most, 100=increase-most
)

const (
	DIRECTION_MAX = 100
	DIRECTION_MIN = 0
)

type MoveOutput struct {
	Status ServiceStatus
	Position uint8 //current position
}

func seatMove(vehicleId string, direction DirectionType, position uint8, moveCallback func(MoveOutput)) MoveOutput {
	var moveOutput MoveOutput
	moveOutput.Status = SUCCESS
	moveOutput.Position = 50
	return moveOutput
}
//Asynchronous move is realized by setting the position to DIRECTION_MAX or DIRECTION_MIN and then cancel the service asynchronously

