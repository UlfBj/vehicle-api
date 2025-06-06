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
	"math/rand"

	"flag"
	"net/url"
	"github.com/gorilla/websocket"
//	"github.com/akamensky/argparse"
//	"github.com/covesa/vissr/utils"
//	"github.com/google/uuid"
)

type Percentage uint8  // max = 100

type MatrixId struct {
	RowName string
	ColumnName string
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

type VehicleAccess struct {
	Service VehicleServices
	Signal SignalServices
}

type SignalServices struct {
	Get func(VehicleHandle, string, string, string) GetOutput
	Set func(VehicleHandle, string, string, string, string)
	Subscribe func(VehicleHandle, string, string, string, func(SubscribeOutput)) SubscribeOutput
	Unsubscribe func(VehicleHandle, ServiceHandle)
}

type VehicleServices struct {
	Seating SeatServices
	HVAC HVACServices
//	ExteriorLighting ExteriorLightingServices
//	InteriorLighting InteriorLightingServices
}

type SeatServices struct {
	GetPropertiesSeatId func(VehicleHandle) GetPropertiesSeatIdOutput
	MoveSeat func(VehicleHandle, MatrixId, string, Percentage, string, func(MoveSeatOutput)) MoveSeatOutput
}

type HVACServices struct {
	Service1 func(VehicleHandle)
}

type ConnectivityData struct {
	IpAddress string
	PortNo string
	Protocol string
}

type ConnectedData struct {
	protocol string
	socket string
	clientTopic string
	wsConn *websocket.Conn
	wsSessions uint8
}

type vehicleInstance struct {
	vehicleGuid string
	vehicleId VehicleHandle
	connectivity []ConnectivityData
	connected ConnectedData
	sessionHandle []uint32
	nextInstance *vehicleInstance
}

type VehicleHandle uint32
type ServiceHandle uint32

type ConnectOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	VehicleId VehicleHandle
	Protocol []string
	LtCredentials string
}

var vehicleList *vehicleInstance

func generateRandomUint32() VehicleHandle {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return (VehicleHandle)(r.Uint32())
}

func generateRandomString() string {
	uint32Topic := generateRandomUint32()
	return fmt.Sprint(uint32Topic)
}

func getSupportedConnectivity(vehicleGuid string) []ConnectivityData { // this method must be implemented to match the "ecosystem requirements"
	support := make([]ConnectivityData, 4) //VISSv3.0 main options. First in list will be set as default.
	support[0].IpAddress = "127.0.0.1"
	support[0].PortNo = "8080"
	support[0].Protocol = "VISSv3.0-ws"
//	support[0].PortNo = "6443"
//	support[0].Protocol = "VISSv3.0-wss"
	support[1].IpAddress = "127.0.0.1"
	support[1].PortNo = "8887"
	support[1].Protocol = "VISSv3.0-grpc"
//	support[1].PortNo = "5443"
//	support[1].Protocol = "VISSv3.0-grpcs"
	support[2].IpAddress = "127.0.0.1"
	support[2].PortNo = "1883"
	support[2].Protocol = "VISSv3.0-mqtt"
//	support[2].PortNo = "8883"
//	support[2].Protocol = "VISSv3.0-mqtts"
	support[3].IpAddress = "127.0.0.1"
	support[3].PortNo = "8888"
	support[3].Protocol = "VISSv3.0-http"
//	support[3].PortNo = "443"
//	support[3].Protocol = "VISSv3.0-https"
	return support
}

func SelectProtocol(vehicleId VehicleHandle, protocol string) {
	vehicle := getConnectInstance(vehicleId)
	if vehicle == nil {
		fmt.Printf("SelectProtocol: Protocol %s not found\n", protocol)
		return
	}
	for i := 0; i<len(vehicle.connectivity); i++ {
		if vehicle.connectivity[i].Protocol == protocol {
			vehicle.connected.protocol = protocol
			vehicle.connected.socket = vehicle.connectivity[i].IpAddress + ":" + vehicle.connectivity[i].PortNo
		}
	}
}

func Connect(vehicleGuid string, clientCredentials string) ConnectOutput {
	var output ConnectOutput
	support := getSupportedConnectivity(vehicleGuid)
	output.VehicleId = generateRandomUint32()
	output.Protocol = make([]string, len(support))
	for i := 0; i < len(support); i++ {
		output.Protocol[i] = support[i].Protocol
	}
	output.LtCredentials = ""  // not implemented
	output.Status = SUCCESSFUL
	var instance vehicleInstance
	instance.vehicleGuid = vehicleGuid
	instance.vehicleId = output.VehicleId
	instance.connectivity = support
	instance.connected.protocol = support[0].Protocol  // set as default
	instance.connected.socket = support[0].IpAddress + ":" + support[0].PortNo  // set as default
	instance.connected.clientTopic = generateRandomString()  //needed for VISSv3.0-mqtt
	addVehicle(&instance)
	return output
}

func (v VehicleHandle) Disconnect(vehicleId VehicleHandle) {
	vehicle := getConnectInstance(vehicleId)
	if vehicle == nil {
		fmt.Printf("Disconnect: vehicleId %s not found\n", vehicleId)
		return
	}
	if vehicle.connected.wsConn != nil {
		vehicle.connected.wsConn.Close()
		vehicle.connected.wsSessions--
		if vehicle.connected.wsSessions == 0{
			vehicle.connected.wsConn = nil
		}
	}
	removeVehicle(vehicleId)
}

func (v VehicleHandle) InitAccess(vehicleId VehicleHandle) *VehicleAccess {
	var va VehicleAccess
	// connect for GetMetadata?
	va.Signal.Get = Get
	va.Signal.Subscribe= Subscribe
	va.Service.Seating.GetPropertiesSeatId = GetPropertiesSeatId
	va.Service.Seating.MoveSeat = MoveSeat
	va.Service.HVAC.Service1 = hvacService1
	return &va
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

func removeVehicle(vehicleId VehicleHandle) {
	if vehicleList == nil {
		return
	} else {
		iterator := &vehicleList
		for *iterator != nil {
			if (*iterator).vehicleId == vehicleId {
				fmt.Printf("Disconnected to vehicle id=%s()\n", (*iterator).vehicleGuid)
				*iterator =(*iterator).nextInstance
				break
			}
			iterator = &(*iterator).nextInstance
		}
	}
}

func isConnected(vehicleId VehicleHandle) bool {
	instance := getConnectInstance(vehicleId)
	if instance == nil {
		return false
	} else {
		return true
	}
}

func GetVehicleName(vehicleId VehicleHandle) string {
	instance := getConnectInstance(vehicleId)
	if instance == nil {
		return ""
	} else {
		return instance.vehicleGuid
	}
}

func getConnectInstance(vehicleId VehicleHandle) *vehicleInstance {
	if vehicleList == nil {
		return nil
	} else {
		iterator := vehicleList
		for iterator != nil {
			if iterator.vehicleId == vehicleId {
				return iterator
			}
			iterator = iterator.nextInstance
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

func sendMessage(vehicle *vehicleInstance, clientMessage string, outChan chan string) string {
	response := ""
	switch vehicle.connected.protocol {
		case "VISSv3.0-wss": fallthrough
		case "VISSv3.0-ws": response = sendMessageWs(vehicle, clientMessage, outChan)
		case "grpc":
		case "mqtt":
		case "http":
	}
	return response
}

func sendMessageWs(vehicle *vehicleInstance, clientMessage string, outChan chan string) string {
	response := ""
	conn := vehicle.connected.wsConn
	if conn == nil {
		conn = initVissV2WebSocket(vehicle.connected.socket)
		vehicle.connected.wsConn = conn
	}
	if conn != nil {
		vehicle.connected.wsSessions++
		response = performWsCommand(clientMessage, conn)
		if !strings.Contains(response, `"error:"`) && outChan != nil {
			go receiveEventsWs(conn, outChan)
		}
	}
	return response
}

func receiveEventsWs(conn *websocket.Conn, outChan chan string) {
	for {
		_, event, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Event error: %s\n", err)
			continue
		}
//		fmt.Printf("Event: %s\n", string(event))
		outChan <- string(event)
	}
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

func performWsCommand(command string, conn *websocket.Conn) string {
//	fmt.Printf("Request: %s\n", command)
	jsonResponse := getWsResponse(conn, []byte(command))
//	fmt.Printf("Response: %s\n", jsonResponse)
/*	if strings.Contains(command, `"subscribe"`) {
		for {
			_, event, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Notification error: %s\n", err)
				return
			}
//			fmt.Printf("Notification: %s\n", string(event))
		}
	}*/
	return jsonResponse
}

func getWsResponse(conn *websocket.Conn, request []byte) string {
	err := conn.WriteMessage(websocket.BinaryMessage, request)
	if err != nil {
		fmt.Printf("Request error:%s\n", err)
		return ""
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		fmt.Printf("Response error: %s\n", err)
		return ""
	}
	return string(msg)
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

//Signal access
func Get(vehicleId VehicleHandle, path string, filter string, stCredentials string) GetOutput {
	connection := getConnectInstance(vehicleId)
	filterParam := ""
	if filter != "" {
		filterParam = `, "filter":` + filter
	}
	stCredParam := ""
	if stCredentials != "" {
		stCredParam = `, "authorization":"` + stCredentials + "\""
	}
	clientMessage := `{"action":"get", "path":"` + path + "\"" + filterParam + stCredParam + `, "requestId":"` + generateRandomString() + `"}`
	serverMessage := sendMessage(connection, clientMessage, nil)
	var out GetOutput
	var outMap map[string]interface{}
	err := json.Unmarshal([]byte(serverMessage), &outMap)
	if err != nil {
		fmt.Printf("Get():unmarshal error=%s", err)
		out.Status = FAILED
		var errorData ErrorData
		errorData.Code = 502
		errorData.Reason = "bad_gateway"
		errorData.Description = "The upstream server response was invalid"
		out.Error = &errorData
		return out
	}
	if outMap["error"] != nil {
		out.Status = FAILED
		out.Error = getErrorInfo(outMap["error"].(map[string]interface{}))
	} else {
		out.Status = SUCCESSFUL
		out.Data = populateData(outMap["data"])
	}
	return out
}

func eventHandler(outChan chan string, callback interface{}) {
	for {
		select {
			case event := <- outChan:
			if event == "VAPI-cancel-session" {
				break
			}
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

func reformatOutput(event string, outputType string) interface{} {
	switch outputType {
		case "subscribe":
			return reformatSubscribeEvent(event)
	}
	return nil
}

func Subscribe(vehicleId VehicleHandle, path string, filter string, stCredentials string, callback func(SubscribeOutput)) SubscribeOutput {
	connection := getConnectInstance(vehicleId)
	filterParam := ""
	if filter != "" {
		filterParam = `, "filter":` + filter
	}
	stCredParam := ""
	if stCredentials != "" {
		stCredParam = `, "authorization":"` + stCredentials + "\""
	}
	clientMessage := `{"action":"subscribe", "path":"` + path + "\"" + filterParam + stCredParam + `, "requestId":"` + generateRandomString() + `"}`
	outChan := make(chan string)
	go eventHandler(outChan, callback)
	serverMessage := sendMessage(connection, clientMessage, outChan)
	return reformatSubscribeEvent(serverMessage)
}

func reformatSubscribeEvent(event string) SubscribeOutput {
	var out SubscribeOutput
	var outMap map[string]interface{}
	err := json.Unmarshal([]byte(event), &outMap)
	if err != nil {
		fmt.Printf("Get():unmarshal error=%s", err)
		out.Status = FAILED
		var errorData ErrorData
		errorData.Code = 502
		errorData.Reason = "bad_gateway"
		errorData.Description = "The upstream server response was invalid"
		out.Error = &errorData
		return out
	}
	if outMap["error"] != nil {
		out.Status = FAILED
		out.Error = getErrorInfo(outMap["error"].(map[string]interface{}))
	} else {
		out.Status = SUCCESSFUL
		out.Data = populateData(outMap["data"])
	}
	return out
}

//Seat Services
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
	SessionHandle int32
}

type MoveSeatOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Position Percentage
}

func MoveSeat(vehicleId VehicleHandle, seatId MatrixId, movementType string, position Percentage, stCredentials string, callback func(MoveSeatOutput)) MoveSeatOutput {
	var out MoveSeatOutput
	out.Status = SUCCESSFUL
	out.Position = 50
	return out
}
//Asynchronous move is realized by setting the position to DIRECTION_MAX or DIRECTION_MIN and then cancel the service asynchronously

type RowDef struct {
	RowName string
	ColumnName []string
}

type RaggedMatrix []RowDef

type GetPropertiesSeatIdOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	SeatIds RaggedMatrix
}

func GetPropertiesSeatId(vehicleId VehicleHandle) GetPropertiesSeatIdOutput {
	var out GetPropertiesSeatIdOutput
	if !isConnected(vehicleId) {
		out.Status = FAILED
		fmt.Printf("hvacService1(): Vehicle is not connected\n")
		return out
	}
	out.Status = SUCCESSFUL
	out.SeatIds = []RowDef{{"Row1", []string{"Left", "Right"}}, {"Row2", []string{"Couch"}}}
	return out
}

// HVAC services
func hvacService1(vehicleId VehicleHandle) {
	if isConnected(vehicleId) {
		fmt.Printf("hvacService1()\n")
	} else {
		fmt.Printf("hvacService1(): Vehicle is not connected\n")
	}
}
