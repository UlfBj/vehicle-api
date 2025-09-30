/**
* (C) 2025 Ford Motor Company
*
* All files and artifacts in the repository at https://github.com/ulfbj/Vehicle-Service-API
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package VapiViss

import (
	"fmt"
	"encoding/json"
//	"io"
//	"io/fs"
//	"net/http"
//	"os"
	"strconv"
	"strings"
	"time"
//	"math"
	"math/rand"

	"flag"
	"net/url"
	"github.com/gorilla/websocket"
//	"github.com/akamensky/argparse"
//	"github.com/covesa/vissr/utils"
//	"github.com/google/uuid"
)

type Percentage float32  // min = 0, max = 100

type MatrixId struct {
	RowName string
	ColumnName string
}

type SeatConfig struct {
	MovementType string
	Position Percentage
}

type ProcedureStatus int8
const (
	ONGOING = 1     // in execution of latest call
	SUCCESSFUL = 0  // terminated successfully in latest call
	FAILED = -1      // terminated due to failure in latest call
)

type ErrorData struct {
	Code int32
	Reason string
	Description string
}

type ConnectivityData struct {
	PortNo string
	Protocol string
}

type ActiveService struct {
	name string
	serviceId uint32
	messageId string
	messageChan chan map[string]interface{}
	cancelChan chan string
	next *ActiveService
}

type ConnectedData struct {
	protocol string
	socket string
	clientTopic string
	connHandle interface{}  //*websocket.Conn, *grpc....
	activeService *ActiveService
	next *ConnectedData
}

type VehicleConnection struct {
	vehicleGuid string
	vehicleId VehicleHandle
	ipAddress string
	connectivitySupport []ConnectivityData
	selectedProtocol string
	connectedData *ConnectedData
	next *VehicleConnection
}

var vehConnList *VehicleConnection
var eventChan chan map[string]interface{}

type VehicleHandle uint32

type GetVehicleOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	VehicleId VehicleHandle
	Protocol []string
}

type ConnectOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	LtCredential string
}

type ServiceSignature struct {
	Name string
	Input string
	Output string
}

type ServiceInquiryOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Service []ServiceSignature
}

type GetStCredentialsOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	StCredentials string
}

type InvokeOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	ServiceOutput string
	ServiceId uint32
}

type GetMetadataOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Metadata string
}

type GeneralOutput struct {
	Status ProcedureStatus
	Error *ErrorData
}

// ****************** Common services ***************
func GetVehicle(vehicleGuid string) GetVehicleOutput {
	var vehConn VehicleConnection
	var out GetVehicleOutput
	vehConn.vehicleGuid = vehicleGuid
	vehConn.connectivitySupport, vehConn.ipAddress = getSupportedConnectivity(vehicleGuid)
	if vehConn.ipAddress == "" {
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "unknown vehicle")
		return out
	}
	vehConn.vehicleId = VehicleHandle(generateRandomUint32())
	addVehicleConnection(&vehConn)
	out.VehicleId = vehConn.vehicleId
	out.Protocol = make([]string, len(vehConn.connectivitySupport))
	for i := 0; i< len(vehConn.connectivitySupport); i++ {
		out.Protocol[i] = vehConn.connectivitySupport[i].Protocol
	}
	out.Status = SUCCESSFUL
	if eventChan == nil {
		eventChan = make(chan map[string]interface{})
		go eventHandler(eventChan)
	}
	return out
}

func ReleaseVehicle(vehicleId VehicleHandle) GeneralOutput {
	var out GeneralOutput
	if vehConnList != nil {
		iterator := &vehConnList
		for *iterator != nil {
			if (*iterator).vehicleId == vehicleId {
				fmt.Printf("Disconnected to vehicle id=%s\n", (*iterator).vehicleGuid)
				*iterator =(*iterator).next
				out.Status = SUCCESSFUL
				return out
			}
			iterator = &(*iterator).next
		}
	}
	out.Status = FAILED
	out.Error = getErrorObject(400, "invalid_data", "unknown vehicle")
	return out
}

func Connect(vehicleId VehicleHandle, protocol string, clientCredentials string) ConnectOutput {
	var out ConnectOutput
	out.LtCredential = ""  // not implemented
	out.Status = SUCCESSFUL
	vehConn := getVehicleConnection(vehicleId)
	if getConnHandle(vehConn.connectedData, protocol) != nil {
		return out
	}
	matchingIndex := -1
	for i := 0; i < len(vehConn.connectivitySupport); i++ {
		if vehConn.connectivitySupport[i].Protocol == protocol {
			matchingIndex = i
			break
		}
	}
	if matchingIndex >= 0 {
		var connectedData ConnectedData
		connectedData.protocol = protocol
		connectedData.socket = vehConn.ipAddress + ":" + vehConn.connectivitySupport[matchingIndex].PortNo
		if strings.Contains(protocol, "mqtt") || strings.Contains(protocol, "MQTT") {
			connectedData.clientTopic = generateRandomString()  //needed for VISSv3.0-mqtt
		}
		connectedData.connHandle = connectToVehicle(protocol, connectedData.socket)
		if connectedData.connHandle != nil {
			addConnectedData(&(vehConn.connectedData), &connectedData)
			vehConn.selectedProtocol = protocol
			go initReceiveMessage(vehConn, protocol)
		} else {
			out.Error = getErrorObject(502, "bad_gateway", "Protocol not supported")
			out.Status = FAILED
		}
	} else {
		out.Error = getErrorObject(400, "invalid_data", "The upstream server response was invalid")
		out.Status = FAILED
	}
	return out
}

func Disconnect(vehicleId VehicleHandle, protocol string) GeneralOutput {
	var out GeneralOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Error = getErrorObject(400, "invalid_data", "Vehicle not connected")
		out.Status = FAILED
		return out
	}
	for {
		serviceId := getActiveServiceId(vehConn.connectedData, protocol)
//fmt.Printf("Disconnect: serviceId = %d\n", serviceId)
		if serviceId == 0 {
			break
		}
		Unsubscribe(vehicleId, serviceId)
		removeActiveService(&vehConn.connectedData, protocol, serviceId)
	}
	removeConnection(&vehConn, protocol)
	out.Status = SUCCESSFUL
	return out
}

func SelectProtocol(vehicleId VehicleHandle, protocol string) GeneralOutput {
	var out GeneralOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn != nil {
		for i := 0; i<len(vehConn.connectivitySupport); i++ {
			if vehConn.connectivitySupport[i].Protocol == protocol {
				if getConnHandle(vehConn.connectedData, protocol) != nil {
					vehConn.selectedProtocol = protocol
					out.Status = SUCCESSFUL
					return out
				}
			}
		}
	}
	out.Error = getErrorObject(400, "invalid_data", "Protocol not connected")
	out.Status = FAILED
	return out
}

func GetMetadata(vehicleId VehicleHandle, path string, stCredentials string) GetMetadataOutput {
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		var out GetMetadataOutput
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	filterParam := `", "filter": {"variant":"metadata", "parameter":"0"}`
	stCredParam := ""
	if stCredentials != "" {
		stCredParam = `, "authorization":"` + stCredentials + "\""
	}
	serviceId := generateRandomUint32()
	requestId := generateRandomString()
	messageChan := addActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId, requestId)
	clientMessage := `{"action":"get", "path":"` + path + filterParam + stCredParam + `, "requestId":"` + requestId + `"}`
//	clientMessage := `{"action":"get", "path":"` + path + "\"" + filterParam + stCredParam + `, "requestId":"` + requestId + `"}`
	sendMessage(vehConn, "", clientMessage)
	responseMap := <- messageChan
	removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
	return reformatOutput(responseMap, "getmetadata").(GetMetadataOutput)
/*	return reformatOutput(responseMap, "get").(GetOutput)
	requestId := generateRandomString()
	clientMessage := `{"action":"get", "path":"` + path + filterParam + stCredParam + `, "requestId":"` + requestId + `"}`
	responseChan := make(chan map[string]interface{})
	ok := saveReturnHandle(&vehConn.connectedData, vehConn.selectedProtocol, "", "", 0, requestId, responseChan, nil)
	if !ok {
		var out GetMetadataOutput
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle connection is lost")
		return out
	}
	sendMessage(vehConn, "", clientMessage)
	var responseMap map[string]interface{}
	select {
		case responseMap = <- responseChan:  //wait for response from receiveMessage
		
	}
	return reformatOutput(responseMap, "getmetadata").(GetMetadataOutput)*/
}

func ServiceInquiry(vehicleId VehicleHandle) ServiceInquiryOutput {
// TODO: getting the relevant metadata from the vehicle about supported services
	var out ServiceInquiryOutput
	out.Status = FAILED
	out.Error = getErrorObject(503, "service_unavailable", "Service not implemented")
	return out
}

func Invoke(vehicleId VehicleHandle, serviceName string, procedureInput string, stCredentials string, callback func(InvokeOutput)) InvokeOutput {
// TODO: invoking the named service
	var out InvokeOutput
	out.Status = FAILED
	out.Error = getErrorObject(503, "service_unavailable", "Service not implemented")
	return out
}

func GetStCredentials(vehicleId VehicleHandle, ltCredentials string, purpose string) GetStCredentialsOutput {
// TODO: calling a authorization sever to get short-term credentials
	var out GetStCredentialsOutput
	out.Status = FAILED
	out.Error = getErrorObject(503, "service_unavailable", "Service not implemented")
	return out
}

// ****************** Signal services ***************
func Set(vehicleId VehicleHandle, path string, value string, stCredentials string) GeneralOutput {
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		var out GeneralOutput
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	if value == "" {
		var out GeneralOutput
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "missing value")
		return out
	}
	stCredParam := ""
	if stCredentials != "" {
		stCredParam = `, "authorization":"` + stCredentials + "\""
	}
	serviceId := generateRandomUint32()
	requestId := generateRandomString()
	messageChan := addActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId, requestId)
	clientMessage := `{"action":"set", "path":"` + path  + `", "value":"` + value + "\"" + stCredParam + `, "requestId":"` + requestId + `"}`
	sendMessage(vehConn, "", clientMessage)
	responseMap := <- messageChan
	removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
	return reformatOutput(responseMap, "set").(GeneralOutput)
}

func Get(vehicleId VehicleHandle, path string, filter string, stCredentials string) GetOutput {
	var out GetOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	filterParam := ""
	if filter != "" {
		filterParam = `, "filter":` + filter
	}
	stCredParam := ""
	if stCredentials != "" {
		stCredParam = `, "authorization":"` + stCredentials + "\""
	}
	serviceId := generateRandomUint32()
	requestId := generateRandomString()
	messageChan := addActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId, requestId)
	clientMessage := `{"action":"get", "path":"` + path + "\"" + filterParam + stCredParam + `, "requestId":"` + requestId + `"}`
	sendMessage(vehConn, "", clientMessage)
	responseMap := <- messageChan
	removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
	return reformatOutput(responseMap, "get").(GetOutput)
}

func Subscribe(vehicleId VehicleHandle, path string, filter string, stCredentials string, callback func(SubscribeOutput)) SubscribeOutput {
	serviceId := generateRandomUint32()
	return subscribeCore(vehicleId, path, "", filter, stCredentials, serviceId, callback)
}

func subscribeCore(vehicleId VehicleHandle, path string, cancelValue string, filter string, stCredentials string, serviceId uint32, callback func(SubscribeOutput)) SubscribeOutput {
	var out SubscribeOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	filterParam := ""
	if filter != "" {
		filterParam = `, "filter":` + filter
	}
	stCredParam := ""
	if stCredentials != "" {
		stCredParam = `, "authorization":"` + stCredentials + "\""
	}
	requestId := generateRandomString()
	messageChan := addActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId, requestId)
	message := `{"action":"subscribe", "path":"` + path + "\"" + filterParam + stCredParam + `, "requestId":"` + requestId + `"}`
	sendMessage(vehConn, "", message)
	messageMap := <- messageChan
	if messageMap["error"] != nil {
		removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
		return reformatOutput(messageMap, "subscribe").(SubscribeOutput)
	}
	cancelChan := make(chan string)
	ok := saveCancelHandle(&vehConn.connectedData, vehConn.selectedProtocol, serviceId, messageMap["subscriptionId"].(string), cancelChan)
	if !ok {
		removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
		out.Status = FAILED
		out.Error = getErrorObject(502, "bad_gateway", "Server internal error")
		return out
	}
	go func() {
		for {
			select {
			case messageMap = <- messageChan:
			messageMap["serviceId"] = serviceId
			out := reformatOutput(messageMap, "subscribe").(SubscribeOutput)
			callback(out)
			if messageMap["error"] != nil {
				removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
				return
			}
			case <- cancelChan:
				removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
				return
			}
		}
	}()
	out.Status = SUCCESSFUL
	out.Data = nil
	out.ServiceId = serviceId
	return out
}

func Unsubscribe(vehicleId VehicleHandle, serviceId uint32) GeneralOutput {
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		var out GeneralOutput
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	protocol := getProtocol(&vehConn.connectedData, serviceId)
	subscriptionId := getCancelData(vehConn.connectedData, protocol, serviceId)
	removeActiveService(&vehConn.connectedData, protocol, serviceId)
	requestId := generateRandomString()
	serviceId = generateRandomUint32()
	clientMessage := `{"action":"unsubscribe", "subscriptionId":"` + subscriptionId + `", "requestId":"` + requestId + `"}`
	responseChan := addActiveService(&vehConn.connectedData, protocol, serviceId, requestId)
	sendMessage(vehConn, "", clientMessage)
	responseMap := <- responseChan
	removeActiveService(&vehConn.connectedData, protocol, serviceId)
	return reformatOutput(responseMap, "unsubscribe").(GeneralOutput)
}

func CancelService(vehicleId VehicleHandle, serviceId uint32) GeneralOutput {
	var out GeneralOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	Unsubscribe(vehicleId, serviceId)
//	removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
/*	subscriptionId := getCancelData(vehConn.connectedData, vehConn.selectedProtocol, serviceId)
	if len(value) == 0 {
		getOut := Get(vehicleId, path, "", "")
		if getOut.Status == SUCCESSFUL {
			value = getOut.Data[0].Dp[0].Value
		}
	}
	if len(value) != 0 {
		unsubOut := Unsubscribe(vehicleId, serviceId)
		if unsubOut.Status == SUCCESSFUL {
			setOut := Set(vehicleId, path, value, "")
			if setOut.Status == SUCCESSFUL {
				return GeneralOutput{SUCCESSFUL, nil}
			}
		}
	}*/
//	var out GeneralOutput
	out.Status = SUCCESSFUL
//	out.Error = getErrorObject(502, "bad_gateway", "Cancelling of service failed")
	return out
}

// ****************** Seat services ***************
// constants for the different seat movement types
const (
	LONGITUDINAL = "longitudinal" //Forward-backward direction of the vehicle
	VERTICAL = "vertical"         // Up-down direction of the vehicle
	BACKREST = "backrest"         // Seat backrest angular
	LUMBAR = "lumbar"             // Seat inflate-deflate lumbar
)

/* constants for asynchronous seat movements; invoke MoveSeat using one of the constants together with its associated movement type,
*  then terminate the movement by invoking CancelService */
const (
	FORWARD = 0            //longitudinal movement
	BACKWARD = 100         //longitudinal movement
	UP = 100               //vertical movement
	DOWN = 0               //vertical movement
	INFLATE = 100          //lumbar movement
	DEFLATE = 0            //lumbar movement
	FORWARD_RECLINE = 0    //backrest movement
	BACKWARD_RECLINE = 100 //backrest movement
)

// constants for massage support
const (
	ROLL = "roll"
	PULSE = "pulse"
	WAVE = "wave"
)

type DataPoint struct {
	Value string
	Timestamp string
}

type DataContainer struct {
	Path string
	Dp []DataPoint
}

type GetOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Data []DataContainer
}

type SubscribeOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Data []DataContainer
	ServiceId uint32
}

type MassageOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	ServiceId uint32
}

type MoveSeatOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Position Percentage
	ServiceId uint32
}

type ConfigureSeatOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Configured []SeatConfig
	Unconfigured []string
	ServiceId uint32
}

type SupportData struct {
	Name string
	Description string
}

type ColumnData struct {
	Name string
	MovementSupport []SupportData
	MassageSupport []SupportData
}

type RowDef struct {
	RowName string
	Column []ColumnData
}

type RaggedMatrix []RowDef

type GetPropertiesSeatingOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Properties RaggedMatrix
}

func MoveSeat(vehicleId VehicleHandle, seatId MatrixId, movementType string, position Percentage, stCredentials string, callback func(MoveSeatOutput)) MoveSeatOutput {
	var out MoveSeatOutput
	var actuatorPath string
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	if position < 0 || position > 100 {
		out.Error = getErrorObject(400, "invalid_data", "position out of range")
		out.Status = FAILED
		return out
	}
	if !checkSupport(seatId, movementType, "move") {
		out.Error = getErrorObject(400, "invalid_data", "Movement type not supported for this seat")
		out.Status = FAILED
		return out
	}
	var A, B Percentage
	switch movementType {
		case LONGITUDINAL:
			actuatorPath = getSeatPositionedPath("Vehicle.Cabin.Seat.RowX.ColumnY.Position", seatId)
			A = 3
			B = 0
//			position = 3 * position  //in millimeter, 300 mm dynamic range??
		case LUMBAR:
			A = 1
			B = 0
			actuatorPath = getSeatPositionedPath("Vehicle.Cabin.Seat.RowX.ColumnY.Backrest.Lumbar.Support", seatId)
		case BACKREST:
			A = 0.9
			B = -45
			actuatorPath = getSeatPositionedPath("Vehicle.Cabin.Seat.RowX.ColumnY.Backrest.Recline", seatId)
//			position = 0.9 * position - 45 //transform position to degrees; f(x) = 0.9*x - 45 f(0)=-45; f(100)=45 ??
		default:
			out.Error = getErrorObject(400, "invalid_data", "unknown movementType")
			out.Status = FAILED
			return out
	}
	position = A * position + B
	posStr := strconv.FormatFloat(float64(position), 'f', -1, 32)
	if movementType == LONGITUDINAL {
		dotIndex := strings.Index(posStr, ".")
		if dotIndex != -1 {
			posStr = posStr[:dotIndex] // remove fraction
		}
	}
	setOut := Set(vehicleId, actuatorPath, posStr, stCredentials)
	if setOut.Status == FAILED {
		out.Status = FAILED
		out.Error = setOut.Error
		return out
	}
	getOut := Get(vehicleId, actuatorPath, "", stCredentials)
	if getOut.Status == FAILED {
		out.Status = FAILED
		out.Error = getOut.Error
		return out
	}
	currPos, _ := strconv.Atoi(getOut.Data[0].Dp[0].Value)
	out.Position = (Percentage(currPos)-B)/A
	out.Status = ONGOING
/*******/
	serviceId := generateRandomUint32()
	out.ServiceId = serviceId
	requestId := generateRandomString()
	filter := `{"variant":"timebased","parameter":{"period":"500"}}`
	filterParam := `, "filter":` + filter
	stCredParam := ""
	if stCredentials != "" {
		stCredParam = `, "authorization":"` + stCredentials + "\""
	}
	message := `{"action":"subscribe", "path":"` + actuatorPath + "\"" + filterParam + stCredParam + `, "requestId":"` + requestId + `"}`
	messageChan := addActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId, requestId)
	sendMessage(vehConn, "", message)
	messageMap := <- messageChan
	if messageMap["error"] != nil {
		removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
		out.Status = FAILED
		out.Error = getErrorInfo(messageMap["error"].(map[string]interface{}))
		return out
	}
	cancelChan := make(chan string)
	ok := saveCancelHandle(&vehConn.connectedData, vehConn.selectedProtocol, serviceId, messageMap["subscriptionId"].(string), cancelChan)
	if !ok {
		removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
		out.Status = FAILED
		out.Error = getErrorObject(502, "bad_gateway", "Server internal error")
		return out
	}
	go func() {
		for {
			select {
			case messageMap = <- messageChan:
				if messageMap["error"] != nil {
					out.Status = FAILED
					out.Error = getErrorInfo(messageMap["error"].(map[string]interface{}))
				} else {
					out.Status = ONGOING
					data := populateData(messageMap["data"])
					currPos, _ := strconv.Atoi(data[0].Dp[0].Value)
					out.Position = (Percentage(currPos)-B)/A
					if out.Position >= 100 {
						out.Status = SUCCESSFUL
					}
				}
				if callback != nil {
					callback(out)
				}
				if messageMap["error"] != nil || out.Status == SUCCESSFUL {
					Unsubscribe(vehicleId, serviceId)
					removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
					return
				}
			case <- cancelChan:
//				removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
				return
			}
		}
	}()
	return out
}

func ConfigureSeat(vehicleId VehicleHandle, seatId MatrixId, configuration []SeatConfig, stCredentials string, callback func(ConfigureSeatOutput)) ConfigureSeatOutput {
	var out ConfigureSeatOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	go func() {
	var out ConfigureSeatOutput
	out.Status = ONGOING
	moveOut := make([]MoveSeatOutput, len(configuration))
	moveSeatCb := func(cbOut MoveSeatOutput) {
		confIndex := findConfIndex(cbOut.ServiceId, moveOut)
		if confIndex != -1 {
			moveOut[confIndex].Status = cbOut.Status
			moveOut[confIndex].Error = cbOut.Error
			moveOut[confIndex].Position = cbOut.Position
		}
	}
	for i := 0; i < len(configuration); i++ {
		if isSupportedMovement(seatId, configuration[i].MovementType) {
			moveOut[i] = MoveSeat(vehicleId, seatId, configuration[i].MovementType, configuration[i].Position, stCredentials, moveSeatCb)
			out.Configured = append(out.Configured, configuration[i])
		} else {
			out.Unconfigured = append(out.Unconfigured, configuration[i].MovementType)
		}
	}
	done := false
	for !done {
		for i :=0; i < len(moveOut); i++ {
			out.Status = SUCCESSFUL
			if moveOut[i].Status == FAILED {
				out.Status = FAILED
				out.Error = moveOut[i].Error
				done = true
				break
			} else if moveOut[i].Status == ONGOING {
				out.Status = ONGOING
				break
			}
		}
		callback(out)
		if out.Status == SUCCESSFUL{
			done = true
		}
		time.Sleep(1 * time.Second)
	}
	}()
	out.Status = ONGOING
	return out
}

func ActivateMassage(vehicleId VehicleHandle, seatId MatrixId, massageType string, intensity Percentage, duration uint32, stCredentials string, callback func(MassageOutput)) MassageOutput {
	var out MassageOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	if intensity < 0 || intensity > 100 {
		out.Error = getErrorObject(400, "invalid_data", "intensity out of range")
		out.Status = FAILED
		return out
	}
	if !checkSupport(seatId, massageType, "massage") {
		out.Error = getErrorObject(400, "invalid_data", "Massage type not supported for this seat")
		out.Status = FAILED
		return out
	}
	massageOnPath := getSeatPositionedPath("Vehicle.Cabin.Seat.RowX.ColumnY.Switch.Massage.IsOn", seatId)
	intensityPath := getSeatPositionedPath("Vehicle.Cabin.Seat.RowX.ColumnY.Switch.Massage.Intensity", seatId)
	massageTypePath := getSeatPositionedPath("Vehicle.Cabin.Seat.RowX.ColumnY.Switch.Massage.MassageType", seatId)

	intensityStr := strconv.FormatFloat(float64(intensity), 'f', -1, 32)
	setOut := Set(vehicleId, intensityPath, intensityStr, stCredentials)
	if setOut.Status == FAILED {
		out.Status = FAILED
		out.Error = setOut.Error
		return out
	}
	setOut = Set(vehicleId, massageTypePath, massageType, stCredentials)
	if setOut.Status == FAILED {
		out.Status = FAILED
		out.Error = setOut.Error
		return out
	}
	setOut = Set(vehicleId, massageOnPath, "true", stCredentials)
	if setOut.Status == FAILED {
		out.Status = FAILED
		out.Error = setOut.Error
		return out
	}
/****************/
	serviceId := generateRandomUint32()
	out.ServiceId = serviceId
	requestId := generateRandomString()
	if duration == 0 || duration > 24 * 3600 {
		duration = 24 * 3600  //24 hours limit
	}
	filter := `{"variant":"timebased","parameter":{"period":"1000"}}`
	filterParam := `, "filter":` + filter
	stCredParam := ""
	if stCredentials != "" {
		stCredParam = `, "authorization":"` + stCredentials + "\""
	}
	message := `{"action":"subscribe", "path":"` + massageOnPath + "\"" + filterParam + stCredParam + `, "requestId":"` + requestId + `"}`
	messageChan := addActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId, requestId)
	sendMessage(vehConn, "", message)
	messageMap := <- messageChan
	if messageMap["error"] != nil {
		removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
		out.Status = FAILED
		out.Error = getErrorInfo(messageMap["error"].(map[string]interface{}))
		return out
	}
	cancelChan := make(chan string)
	ok := saveCancelHandle(&vehConn.connectedData, vehConn.selectedProtocol, serviceId, messageMap["subscriptionId"].(string), cancelChan)
	if !ok {
		removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
		out.Status = FAILED
		out.Error = getErrorObject(502, "bad_gateway", "Server internal error")
		return out
	}
	go func() {
		finalTime := time.Now().Add(time.Duration(float64(duration)*1e9))
		for {
			select {
			case messageMap = <- messageChan:
				if messageMap["error"] != nil {
					out.Status = FAILED
					out.Error = getErrorInfo(messageMap["error"].(map[string]interface{}))
				} else {
					out.Status = ONGOING
					data := populateData(messageMap["data"])
					massageOn := data[0].Dp[0].Value
					if massageOn == "false" || time.Now().After(finalTime) {
						out.Status = SUCCESSFUL
					}
				}
				if callback != nil {
					callback(out)
				}
				if messageMap["error"] != nil || out.Status == SUCCESSFUL {
					Unsubscribe(vehicleId, serviceId)
//					removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
					return
				}
			case <- cancelChan:
//				removeActiveService(&vehConn.connectedData, vehConn.selectedProtocol, serviceId)
				return
			}
		}
	}()
/*	if callback != nil {
		serviceId := generateRandomUint32()
		if duration == 0 || duration > 24 * 3600 {
			duration = 24 * 3600  //24 hours limit
		}
		callbackInterceptor := makeCallbackInterceptorDuration(vehicleId, callback, serviceId, time.Now().Add(time.Duration(float64(duration)*1e9)))
		filter := `{"variant":"timebased","parameter":{"period":"1000"}}`
		subOut := subscribeCore(vehicleId, massageOnPath, "false", filter, stCredentials, serviceId, callbackInterceptor)
		if subOut.Status == SUCCESSFUL {
			out.Status = ONGOING
			out.ServiceId = serviceId
		} else {
			out.Status = FAILED
			out.Error = getErrorObject(400, "invalid_data", "callback init failed")
		}
	}*/
	return out
}

func GetPropertiesSeating(vehicleId VehicleHandle) GetPropertiesSeatingOutput {
	var out GetPropertiesSeatingOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Error = getErrorObject(400, "invalid_data", "unknown vehicle")
		out.Status = FAILED
		return out
	}
	if len(vehConn.selectedProtocol) == 0 {
		out.Error = getErrorObject(400, "invalid_data", "vehicle not connected")
		out.Status = FAILED
		return out
	}
	out.Status = SUCCESSFUL
	out.Properties = getSimulatedProperties()
	return out
}

// HVAC services
func HvacService1(vehicleId VehicleHandle) GeneralOutput {
	var out GeneralOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Error = getErrorObject(400, "invalid_data", "Protocol not connected")
		out.Status = FAILED
		return out
	}
	out.Status = SUCCESSFUL
	fmt.Printf("HvacService1:succefully simulated")
	return out
}

/****************************** internal functions ************************************************/
func generateRandomUint32() uint32 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Uint32()
}

func generateRandomString() string {
	uint32Topic := generateRandomUint32()
	return fmt.Sprint(uint32Topic)
}

func getSupportedConnectivity(vehicleGuid string) ([]ConnectivityData, string) { // this method must be implemented to match the "ecosystem requirements"
	var ipAddress string
	var support []ConnectivityData
	switch vehicleGuid {
	case "pseudoVin1":
		ipAddress = "127.0.0.1"
		support = make([]ConnectivityData, 4) //VISSv3.0 main options. First in list will be set as default.
		isSecureProtocol := false
		if isSecureProtocol {
			support[0].Protocol = "VISSv3.0-wss"
			support[0].PortNo = "6443"
		} else {
			support[0].Protocol = "VISSv3.0-ws"
			support[0].PortNo = "8080"
		}
		if isSecureProtocol {
			support[1].Protocol = "VISSv3.0-grpcs"
			support[1].PortNo = "5443"
		} else {
			support[1].Protocol = "VISSv3.0-grpc"
			support[1].PortNo = "8887"
		}
		if isSecureProtocol {
			support[2].Protocol = "VISSv3.0-mqtts"
			support[2].PortNo = "8883"
		} else {
			support[2].Protocol = "VISSv3.0-mqtt"
			support[2].PortNo = "1883"
		}
		if isSecureProtocol {
			support[3].Protocol = "VISSv3.0-https"
			support[3].PortNo = "443"
		} else {
			support[3].Protocol = "VISSv3.0-http"
			support[3].PortNo = "8888"
		}
	case "pseudoVin2":
		ipAddress = "192.168.1.247"
		support = make([]ConnectivityData, 1) //VISSv3.0 main options. First in list will be set as default.
		isSecureProtocol := false
		if isSecureProtocol {
			support[0].Protocol = "VISSv3.0-grpcs"
			support[0].PortNo = "5443"
		} else {
			support[0].Protocol = "VISSv3.0-grpc"
			support[0].PortNo = "8887"
		}
	}
	return support, ipAddress
}

func getErrorObject(code int32, reason string, description string) *ErrorData {
		var errData ErrorData
		errData.Code = code
		errData.Reason = reason
		errData.Description = description
		return &errData
}

func addConnectedData(connectedDataList **ConnectedData, connectedData *ConnectedData) {
	if *connectedDataList == nil {
		*connectedDataList = connectedData
	} else {
		iterator := connectedDataList
		for (*iterator).next != nil {
			iterator = &(*iterator).next
		}
		(*iterator).next = connectedData
	}
}

func removeConnection(vehConn **VehicleConnection, protocol string) {
	connectedDataList := &(*vehConn).connectedData
	if *connectedDataList == nil {
		return
	} else {
		iterator := connectedDataList
		for *iterator != nil {
			if (*iterator).protocol == protocol {
//fmt.Printf("removeConnection: removed\n")
				if (*vehConn).selectedProtocol == protocol {
					(*vehConn).selectedProtocol = ""
				}
				closeConnection((*iterator).connHandle, protocol)
				*iterator =(*iterator).next
				break
			}
			iterator = &(*iterator).next
		}
	}
}

func closeConnection(connHandle interface{}, protocol string) {
	switch protocol {
		case "VISSv3.0-wss": fallthrough
		case "VISSv3.0-ws":
			connHandle.(*websocket.Conn).Close()
		case "grpc":
		case "mqtt":
		case "http":
		default: fmt.Printf("%s is unsupportd protocol\n", protocol)
	}
}

/*
func removeSession(connectedDataList **ConnectedData, protocol string) {
	if *connectedDataList == nil {
		return
	} else {
		iterator := connectedDataList
		for *iterator != nil {
			if (*iterator).protocol == protocol {
fmt.Printf("removeSession: removed\n")
				*iterator =(*iterator).next
				break
			}
			iterator = &(*iterator).next
		}
	}
}*/
/*
func getResponseChan(connectedDataList *ConnectedData, protocol string) chan map[string]interface{} {
	if connectedDataList == nil {
		return nil
	} else {
		iterator := connectedDataList
		for iterator != nil {
			if iterator.protocol == protocol {
				return iterator.responseChan
			}
			iterator = iterator.next
		}
	}
	return nil
}*/

/*func getResponseChan(connectedDataList *ConnectedData, messageId string) chan map[string]interface{} {
	if connectedDataList == nil {
		return nil
	} else {
		iterator := connectedDataList
		for iterator != nil {
			activeServiceIterator := iterator.activeService
			if activeServiceIterator == nil {
				continue
			}
			for activeServiceIterator != nil {
				if activeServiceIterator.messageId == messageId {
fmt.Printf("getResponseChan: activeService.messageId=%s\n", messageId)
					return activeServiceIterator.responseChan
				}
				activeServiceIterator = activeServiceIterator.next
			}
			iterator = iterator.next
		}
	}
	return nil
}*/

func removeActiveService(connectedDataList **ConnectedData, protocol string, serviceId uint32) {
	if *connectedDataList == nil {
		fmt.Printf("removeActiveService: connectedDataList is empty for protocol=%s, serviceId=%s\n", protocol, serviceId) // should not be possible...
		return
	} else {
		iterator := connectedDataList
		for *iterator != nil {
			if (*iterator).protocol == protocol {
				activeServiceIterator := &(*iterator).activeService
				for *activeServiceIterator != nil {
					if (*activeServiceIterator).serviceId == serviceId {
//fmt.Printf("removeActiveService: removed\n")
						*activeServiceIterator = nil
						return
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
			}
			iterator = &(*iterator).next
		}
//fmt.Printf("removeActiveService: %d not found\n", serviceId)
	}
}
/*
func updateActiveServiceKey(connectedDataList **ConnectedData, protocol string, requestId string, subscriptionId string) {
	if *connectedDataList == nil {
		fmt.Printf("updateActiveServiceKey: connectedDataList is empty for protocol=%s, requestId=%s\n", protocol, requestId) // should not be possible...
		return
	} else {
		iterator := connectedDataList
		for *iterator != nil {
			if (*iterator).protocol == protocol {
				activeServiceIterator := &(*iterator).activeService
				for *activeServiceIterator != nil {
					if (*activeServiceIterator).subscriptionId == requestId {
						(*activeServiceIterator).subscriptionId = subscriptionId
//fmt.Printf("updateActiveServiceKey: updated key %s->%s\n", requestId, subscriptionId)
						return
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
			}
			iterator = &(*iterator).next
		}
	}
}*/

func getActiveServiceId(connectedDataList *ConnectedData, protocol string) uint32 {
	if connectedDataList == nil {
		return 0
	} else {
		iterator := connectedDataList
		for iterator != nil {
			if (*iterator).protocol == protocol {
				activeServiceIterator := (*iterator).activeService
				if activeServiceIterator != nil {
					return (*activeServiceIterator).serviceId
				}
			}
			iterator = (*iterator).next
		}
	}
//	fmt.Printf("getActiveServiceId: no match for protocol=%s\n", protocol)
	return 0
}

func getCancelData(connectedDataList *ConnectedData, protocol string, serviceId uint32) string {
	if connectedDataList == nil {
		fmt.Printf("getCancelData: connectedDataList is empty for protocol=%s, serviceId=%d\n", protocol, serviceId)
		return ""
	} else {
		iterator := connectedDataList
		for iterator != nil {
			if (*iterator).protocol == protocol {
				activeServiceIterator := (*iterator).activeService
				for activeServiceIterator != nil {
					if (*activeServiceIterator).serviceId == serviceId {
						return (*activeServiceIterator).messageId
					}
					activeServiceIterator = (*activeServiceIterator).next
				}
			}
			iterator = (*iterator).next
		}
	}
	return ""
}

func getMessageChan(connectedDataList *ConnectedData, protocol string, messageId string) chan map[string]interface{} {
	if connectedDataList == nil {
		fmt.Printf("getMessageChan: connectedDataList is empty for protocol=%s, messageId=%s\n", protocol, messageId)
		return nil
	} else {
		iterator := connectedDataList
		for iterator != nil {
			if (*iterator).protocol == protocol {
				activeServiceIterator := &(*iterator).activeService
				for *activeServiceIterator != nil {
					if (*activeServiceIterator).messageId == messageId {
						return (*activeServiceIterator).messageChan
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
			}
			iterator = (*iterator).next
		}
	}
	fmt.Printf("getMessageChan: no match for protocol=%s, messageId=%s\n", protocol, messageId)
	return nil
}

func getProtocol(connectedDataList **ConnectedData, serviceId uint32) string {
	if *connectedDataList == nil {
		return ""
	} else {
		iterator := connectedDataList
		for *iterator != nil {
//			if (*iterator).protocol == protocol {
				activeServiceIterator := &(*iterator).activeService
				for *activeServiceIterator != nil {
					if (*activeServiceIterator).serviceId == serviceId {
//						(*activeServiceIterator).messageId = subscriptionId
//						(*activeServiceIterator).cancelChan = cancelChan
						return (*iterator).protocol
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
//			}
			iterator = &(*iterator).next
		}
	}
	return ""
}

func saveCancelHandle(connectedDataList **ConnectedData, protocol string, serviceId uint32, subscriptionId string, cancelChan chan string) bool {
	if *connectedDataList == nil {
		return false
	} else {
		iterator := connectedDataList
		for *iterator != nil {
			if (*iterator).protocol == protocol {
				activeServiceIterator := &(*iterator).activeService
				for *activeServiceIterator != nil {
					if (*activeServiceIterator).serviceId == serviceId {
						(*activeServiceIterator).messageId = subscriptionId
						(*activeServiceIterator).cancelChan = cancelChan
						return true
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
			}
			iterator = &(*iterator).next
		}
	}
	return false
}

func addActiveService(connectedDataList **ConnectedData, protocol string, serviceId uint32, messageId string) chan map[string]interface{} {
	messageChan := make(chan map[string]interface{})
	if *connectedDataList == nil {
		return nil
	} else {
		iterator := connectedDataList
		for *iterator != nil {
			if (*iterator).protocol == protocol {
				activeServiceIterator := &(*iterator).activeService
				if *activeServiceIterator == nil {
					var activeService ActiveService
					activeService.messageChan = messageChan
					activeService.serviceId = serviceId
					activeService.messageId = messageId
					*activeServiceIterator = &activeService
					return messageChan
				}
				for *activeServiceIterator != nil {
					if (*activeServiceIterator).next == nil {
						var activeService ActiveService
						activeService.messageChan = messageChan
						activeService.serviceId = serviceId
						activeService.messageId = messageId
						(*activeServiceIterator).next = &activeService
						return messageChan
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
			}
			iterator = &(*iterator).next
		}
	}
	return nil
}
/*
func saveReturnHandle(connectedDataList **ConnectedData, protocol string, path string, cancelValue string, serviceId uint32, requestId string, responseChan chan map[string]interface{}, callback interface{}) bool {
	if *connectedDataList == nil {
		return false
	} else {
		iterator := connectedDataList
		for *iterator != nil {
			if (*iterator).protocol == protocol {
				(*iterator).responseChan = responseChan
				activeServiceIterator := &(*iterator).activeService
				if *activeServiceIterator == nil {
					var activeService ActiveService
					activeService.path = path
					activeService.value = cancelValue
					activeService.messageId = requestId
					activeService.serviceId = serviceId
					activeService.callback = callback
					*activeServiceIterator = &activeService
					return true
				}
				for *activeServiceIterator != nil {
					if (*activeServiceIterator).next == nil {
						var activeService ActiveService
						activeService.path = path
						activeService.value = cancelValue
						activeService.messageId = requestId
						activeService.serviceId = serviceId
						activeService.callback = callback
						(*activeServiceIterator).next = &activeService
						return true
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
			}
			iterator = &(*iterator).next
		}
	}
	return false
}*/

func getConnHandle(connectedDataList *ConnectedData, protocol string) interface{} {
	if connectedDataList == nil {
		return nil
	} else {
		iterator := connectedDataList
		for iterator != nil {
//fmt.Printf("getConnHandle: iterator.protocol=%s\n", iterator.protocol)
			if iterator.protocol == protocol {
				if strings.Contains(protocol, "ws") {
					return iterator.connHandle
				}
			}
			iterator = iterator.next
		}
	}
	return nil
}

func addVehicleConnection(vehConn *VehicleConnection) {
	if vehConnList == nil {
		vehConnList = vehConn
	} else {
		iterator := vehConnList
		for iterator.next != nil {
			iterator = iterator.next
		}
		iterator.next = vehConn
	}
}

func getVehicleConnection(vehicleId VehicleHandle) *VehicleConnection {
	if vehConnList == nil {
		return nil
	} else {
		iterator := vehConnList
		for iterator != nil {
			if iterator.vehicleId == vehicleId {
				return iterator
			}
			iterator = iterator.next
		}
	}
	return nil
}

func extractErrorInfo(infoType string, serverMessage string) string {
	switch infoType {
		case "number":
		case "reason":
		case "description":
	}
	offset := strings.Index(serverMessage, "\"" + infoType + "\":")
	if offset == -1 {
		return ""
	}
	firstQuoteIndex := strings.Index(serverMessage[offset:], "\"")
	secondQuoteIndex := strings.Index(serverMessage[offset+firstQuoteIndex+1:], "\"")
fmt.Printf("extractErrorInfo: infoType= %s, info=%s\n", infoType, serverMessage[offset+firstQuoteIndex+1:offset+firstQuoteIndex+1+secondQuoteIndex])
	return serverMessage[offset+firstQuoteIndex+1:offset+firstQuoteIndex+1+secondQuoteIndex]
}

func getErrorInfo(errorMap map[string]interface{}) *ErrorData {
	var errorInfo ErrorData
	for k, v := range errorMap {
//		Info.Println("key=",k, "v=", v)
		if k == "number" {
			code, _ := strconv.Atoi(v.(string))
			errorInfo.Code = (int32)(code)
		}
		if k == "reason" {
			errorInfo.Reason = v.(string)
		}
		if k == "description" {
			errorInfo.Description = v.(string)
		}
	}
	return &errorInfo
}

func sendMessage(vehicle *VehicleConnection, protocol string, clientMessage string) {
	if len(protocol) == 0 {
		protocol = vehicle.selectedProtocol
	}
	switch protocol {
		case "VISSv3.0-wss": fallthrough
		case "VISSv3.0-ws":
			conn := getConnHandle(vehicle.connectedData, "VISSv3.0-ws").(*websocket.Conn)
			sendMessageWs(conn, clientMessage)
		case "grpc":
		case "mqtt":
		case "http":
//		default: response =  `{"error": {"number": "502", "reason": "bad_gateway", "description": "The active protocol is not supported."}}`
	}
}

func sendMessageWs(conn *websocket.Conn, clientMessage string) {
	err := conn.WriteMessage(websocket.BinaryMessage, []byte(clientMessage))
	if err != nil {
		fmt.Printf("Request error:%s\n", err)
	}
}

func initReceiveMessage(vehicle *VehicleConnection, protocol string) {
	if len(protocol) == 0 {
		protocol = vehicle.selectedProtocol
	}
	switch protocol {
		case "VISSv3.0-wss": fallthrough
		case "VISSv3.0-ws":
			for{
				conn := getConnHandle(vehicle.connectedData, "VISSv3.0-ws").(*websocket.Conn)
				if conn == nil {
//fmt.Printf("receiveMessageWs: terminating\n")
					return
				}
				_, message, err := conn.ReadMessage()
				if err != nil {
//fmt.Printf("receiveMessageWs: terminating\n")
					return
				}
				fmt.Printf("receiveMessageWs: message=%s\n", string(message))
				var messageMap map[string]interface{}
				err = json.Unmarshal(message, &messageMap)
				if err != nil {
					fmt.Printf("initReceiveMessage:error message=%s, err=%s", message, err)
					continue
				}
				messageId := extractMessageId(messageMap)
				messageChan := getMessageChan(vehicle.connectedData, protocol, messageId)
				if messageChan != nil {
					messageChan <- messageMap
				}
			}
		case "grpc": //TBI
		case "mqtt": //TBI
		case "http": //TBI
	}
}

func extractMessageId(messageMap map[string]interface{}) string {
	if messageMap["requestId"] != nil {
		return messageMap["requestId"].(string)
	}
	if messageMap["subscriptionId"] != nil {
		return messageMap["subscriptionId"].(string)
	}
	return ""	
}

/*func receiveMessageWs(vehicleId VehicleHandle, conn *websocket.Conn, eventChan chan map[string]interface{}) {
	var responseChan chan map[string]interface{}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			select {
				case <- responseChan:  // terminate message shall be the only possibility...
//fmt.Printf("receiveMessageWs: terminating\n")
				return
				default:
					if !strings.Contains(err.Error(), "use of closed network connection") { // result of Disconnect/Unsubscribe...
						fmt.Printf("Server communication error: %s\n", err)
					}
					time.Sleep(1 * time.Second)
					continue
			}
		}
fmt.Printf("receiveMessageWs: message=%s\n", string(message))
		var messageMap map[string]interface{}
		err = json.Unmarshal(message, &messageMap)
		if err != nil {
			fmt.Printf("receiveMessageWs:error message=%s, err=%s", string(message), err)
			continue
		}
		vehConn := getVehicleConnection(vehicleId)
		protocol := vehConn.selectedProtocol
		requestId, subscriptionId := getMessageId(messageMap)
		if len(requestId) > 0 {
			responseChan = getResponseChan(vehConn.connectedData, protocol)
			if len(subscriptionId) == 0 { // response
				removeActiveService(&vehConn.connectedData, protocol, requestId)
			} else { // subscribe response
				updateActiveServiceKey(&vehConn.connectedData, protocol, requestId, subscriptionId)
			} 
//			fmt.Printf("Response: %s\n", string(message))
			responseChan <- messageMap
		} else if len(subscriptionId) > 0 { // subscription event
//			fmt.Printf("Event: %s\n", string(message))
			eventChan <- messageMap
		}
	}
}*/

func getMessageId(messageMap map[string]interface{}) (string, string) {
	requestId := ""
	subscriptionId := ""
	if messageMap["requestId"] != nil {
		requestId = messageMap["requestId"].(string)
	}
	if messageMap["subscriptionId"] != nil {
		subscriptionId = messageMap["subscriptionId"].(string)
	}
	return requestId, subscriptionId
}

func initVissV2WebSocket(socket string) *websocket.Conn {
	scheme := "ws"
/*	portNum := "8080"
	if secConfig.TransportSec == "yes" {
		scheme = "wss"
		portNum = secConfig.WsSecPort
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{
			Certificates: []tls.Certificate{clientCert},
			RootCAs:      &caCertPool,
		}
	}*/
	var addr = flag.String("addr", socket, "http service address")
	dataSessionUrl := url.URL{Scheme: scheme, Host: *addr, Path: ""}
	subProtocol := make([]string, 1)
	subProtocol[0] = "VISSv2"
	dialer := websocket.Dialer{
		HandshakeTimeout: time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		Subprotocols:     subProtocol,
	}
	conn, _, err := dialer.Dial(dataSessionUrl.String(), nil)
	if err != nil {
		fmt.Printf("Data session dial error:%s\n", err)
	}
	return conn
}

func populateData(dataMap interface{}) []DataContainer {
	var data []DataContainer
	switch vv := dataMap.(type) {
	case []interface{}:
		data = make([]DataContainer, len(vv))
		for i := 0; i < len(vv); i++ {
			data[i] = populateDataL2(vv[i].(map[string]interface{}))
		}
	case map[string]interface{}:
		data = make([]DataContainer, 1)
		data[0] = populateDataL2(vv)
	case nil: // subscribe response
	default:
		fmt.Println("populateData():unknown output type=", vv)
	}

	return data
}

func populateDataL2(dataMap map[string]interface{}) DataContainer {
	var data DataContainer
	for k, v := range dataMap {
//		fmt.Println("key=",k, "v=", v)
		if k == "path" {
			data.Path = v.(string)
		}
		if k == "dp" {
			data.Dp = populateDp(v.(interface{}))
		}
	}
	return data
}

func populateDp(dpMap interface{}) []DataPoint {
	var dp []DataPoint
	switch vv := dpMap.(type) {
	case []interface{}:
		dp = make([]DataPoint, len(vv))
		for i := 0; i < len(vv); i++ {
			dp[i] = populateDpL2(vv[i].(map[string]interface{}))
		}
	case map[string]interface{}:
		dp = make([]DataPoint, 1)
		dp[0] = populateDpL2(vv)
	default:
		fmt.Println("populateDp():unknown output type=", vv)
	}

	return dp
}

func populateDpL2(dpMap map[string]interface{}) DataPoint {
	var dp DataPoint
	for k, v := range dpMap {
//		fmt.Println("key=",k, "v=", v)
		if k == "ts" {
			dp.Timestamp = v.(string)
		}
		if k == "value" {
			dp.Value = v.(string)
		}
	}
	return dp
}

func eventHandler(eventChan chan map[string]interface{}) {
	for {
		select {
			case event := <- eventChan:
			if event["error"] != nil && event["error"] == "VAPI-cancel-session" {
				break
			}
			callback := getCallback(event["subscriptionId"].(string))
			switch vv := callback.(type) {
				case func(SubscribeOutput):
					out := reformatOutput(event, "subscribe").(SubscribeOutput)
					callback.(func(SubscribeOutput))(out)
				default:
					fmt.Println("eventHandler():unknown output type=", vv)
			}
		}
	}
}

func getCallback(subcriptionId string) interface{} {
/*	if vehConnList == nil {
		return nil
	} else {
		iterator := vehConnList
		for iterator != nil {
			if iterator.connectedData == nil {
				return nil
			} else {
				connectedIterator := iterator.connectedData
				for connectedIterator != nil {
					if connectedIterator.activeService == nil {
						return nil
					} else {
						serviceIterator := connectedIterator.activeService
						for serviceIterator != nil {
							if serviceIterator.messageId == subcriptionId {
								return serviceIterator.callback
							}
							serviceIterator = serviceIterator.next
						}
					}
					connectedIterator = connectedIterator.next
				}
			}
			iterator = iterator.next
		}
	}*/
	return nil
}

func reformatOutput(messageMap map[string]interface{}, outputType string) interface{} {
	switch outputType {
		case "set":
			return reformatGeneralMessage(messageMap)
		case "get":
			return reformatGetMessage(messageMap)
		case "getmetadata":
			return reformatGetMetadataMessage(messageMap)
		case "subscribe":
			return reformatSubscribeMessage(messageMap)
		case "unsubscribe": fallthrough
		case "cancelservice":
			return reformatGeneralMessage(messageMap)
	}
	return nil
}

func reformatGetMessage(messageMap map[string]interface{}) GetOutput {
	var out GetOutput
	if messageMap["error"] != nil {
		out.Status = FAILED
		out.Error = getErrorInfo(messageMap["error"].(map[string]interface{}))
	} else {
		out.Status = SUCCESSFUL
		out.Data = populateData(messageMap["data"])
	}
	return out
}

func reformatGetMetadataMessage(messageMap map[string]interface{}) GetMetadataOutput {
	var out GetMetadataOutput
	if messageMap["error"] != nil {
		out.Status = FAILED
		out.Error = getErrorInfo(messageMap["error"].(map[string]interface{}))
	} else {
		out.Status = SUCCESSFUL
		out.Metadata = messageMap["metadata"].(string)
	}
	return out
}

func reformatSubscribeMessage(messageMap map[string]interface{}) SubscribeOutput {
	var out SubscribeOutput
	if messageMap["error"] != nil {
		out.Status = FAILED
		out.Error = getErrorInfo(messageMap["error"].(map[string]interface{}))
	} else {
		out.Status = SUCCESSFUL
		out.Data = populateData(messageMap["data"])
		if messageMap["serviceId"] != nil{
			out.ServiceId = messageMap["serviceId"].(uint32)
		}
	}
	return out
}

func reformatGeneralMessage(messageMap map[string]interface{}) GeneralOutput {
	var out GeneralOutput
	if messageMap["error"] != nil {
		out.Status = FAILED
		out.Error = getErrorInfo(messageMap["error"].(map[string]interface{}))
	} else {
		out.Status = SUCCESSFUL
	}
	return out
}

func connectToVehicle(protocol string, socket string) interface{} {
//fmt.Printf("Socket=%s\n", socket)
	if strings.Contains(protocol, "ws") {
		return initVissV2WebSocket(socket)  // TODO: switch on protocol
	} else if strings.Contains(protocol, "grpc") {
		return nil //not yet implemented
	}
	return nil
}

func getSeatPositionedPath(unpositionedPath string, seatId MatrixId) string { // RowX and ColumnY to be replaced
	index := strings.Index(unpositionedPath, ".RowX.")
	positionedPath := unpositionedPath[:index+1] + seatId.RowName + unpositionedPath[index+1+4:]
	index = strings.Index(positionedPath, ".ColumnY.")
	positionedPath = positionedPath[:index+1] + seatId.ColumnName + positionedPath[index+1+7:]
	return positionedPath
}

func makeCallbackInterceptor(vehicleId VehicleHandle, callback interface{}, serviceId uint32, path string, finalValue string) func(SubscribeOutput) {
	return func(subOut SubscribeOutput) {
		var status ProcedureStatus
		var errorData *ErrorData
		var dataIndex int
		if subOut.Status == SUCCESSFUL {
			status = ONGOING
			for i := 0; i < len(subOut.Data); i++ {
				if subOut.Data[i].Path == path {
					dataIndex = i
					if subOut.Data[i].Dp[0].Value == finalValue {
						status = SUCCESSFUL
						Unsubscribe(vehicleId, serviceId)
						break
					}
				}
			}
		} else {
			status = FAILED
			errorData = subOut.Error
			Unsubscribe(vehicleId, serviceId)
		}
		if callback != nil {
			switch callback.(type) {
				case func(MoveSeatOutput):
					var out MoveSeatOutput
					out.Status = status
					out.Error = errorData
					out.ServiceId = serviceId
					if status != FAILED {
						position, _ := strconv.Atoi(subOut.Data[dataIndex].Dp[0].Value)
						out.Position = Percentage(position)
					}
					callback.(func(MoveSeatOutput))(out)
			}
		}
	}
}

func makeCallbackInterceptorDuration(vehicleId VehicleHandle, callback interface{}, serviceId uint32, finalTime time.Time) func(SubscribeOutput) {
	return func(subOut SubscribeOutput) {
		var status ProcedureStatus
		var errorData *ErrorData
		if subOut.Status == SUCCESSFUL {
			status = ONGOING
			if time.Now().After(finalTime) {
				status = SUCCESSFUL
				Unsubscribe(vehicleId, serviceId)
			}
		} else {
			status = FAILED
			errorData = subOut.Error
			Unsubscribe(vehicleId, serviceId)
		}
		if callback != nil {
			switch callback.(type) {
				case func(MassageOutput):
					var out MassageOutput
					out.Status = status
					out.Error = errorData
					out.ServiceId = serviceId
					callback.(func(MassageOutput))(out)
			}
		}
	}
}

func getSimulatedProperties() RaggedMatrix {
	var properties RaggedMatrix
	properties = make([]RowDef, 2)
	var numofcols []int = []int{2, 2}
	var columnName []string = []string{"DriverSide", "PassengerSide"}
	var movementSupport []SupportData = []SupportData{{LONGITUDINAL, "Seat movement in the direction parallel to the driving direction"},
	{VERTICAL, "Seat movement in the vertical direction to the horizontal plane"}, {LUMBAR, "Seat movement of the lumbar support"}}
	var massageSupport []SupportData = []SupportData{{ROLL, "A rolling massage sensation"},
	{PULSE, "A pulsating massage sensation"}, {WAVE, "A wave like massage sensation"}}
	for i := 0; i < 2; i++ {
		properties[i].RowName = "Row" + strconv.Itoa(i+1)
		properties[i].Column = make([]ColumnData, numofcols[i])
		for j := 0; j < numofcols[i]; j++ {
			properties[i].Column[j].Name = columnName[j]
			properties[i].Column[j].MovementSupport = getSimulatedSupport(i, j, movementSupport)
			properties[i].Column[j].MassageSupport = getSimulatedSupport(i, j, massageSupport)
		}
	}
	return properties
}

func getSimulatedSupport(row int, column int, support []SupportData) []SupportData {
	if row == 0 && column == 0 { // driver
		return support
	} else if row == 0 && column == 1 { // front row passenger
		simSupport := make([]SupportData, 1)
		simSupport[0].Name = support[0].Name
		simSupport[0].Description = support[0].Description
		return simSupport
	}
	return nil // all other passengers
}

func checkSupport(seatId MatrixId, support string, supportType string) bool {
	simProp := getSimulatedProperties()
	switch supportType {
		case "massage":
			for i := 0; i < len(simProp); i++ {
				if simProp[i].RowName == seatId.RowName {
					for j := 0; j < len(simProp[i].Column); j++ {
						if simProp[i].Column[j].Name == seatId.ColumnName {
							for k := 0; k < len(simProp[i].Column[j].MassageSupport); k++ {
								if simProp[i].Column[j].MassageSupport[k].Name == support {
									return true
								}
							}
						}
					}
				}
			}
		case "move":
			for i := 0; i < len(simProp); i++ {
				if simProp[i].RowName == seatId.RowName {
					for j := 0; j < len(simProp[i].Column); j++ {
						if simProp[i].Column[j].Name == seatId.ColumnName {
							for k := 0; k < len(simProp[i].Column[j].MovementSupport); k++ {
								if simProp[i].Column[j].MovementSupport[k].Name == support {
									return true
								}
							}
						}
					}
				}
			}
	}
	return false
}

func isSupportedMovement(seatId MatrixId, movementType string) bool {
	return checkSupport(seatId, movementType, "move")
}

func findConfIndex(serviceId uint32, moveOut []MoveSeatOutput) int {
	for i := 0; i < len(moveOut); i++ {
		if serviceId == moveOut[i].ServiceId {
			return i
		}
	}
	return -1
}
