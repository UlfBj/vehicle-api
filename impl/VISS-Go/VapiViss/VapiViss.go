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
	"math"
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

/*type VehicleAccess struct {
	Service VehicleServices
	Signal SignalServices
}

type SignalServices struct {
	Get func(VehicleHandle, string, string, string) GetOutput
	Set func(VehicleHandle, string, string, string, string) GeneralOutput
	Subscribe func(VehicleHandle, string, string, string, func(SubscribeOutput)) SubscribeOutput
	Unsubscribe func(VehicleHandle, uint32) GeneralOutput
}

type VehicleServices struct {
	Seating SeatServices
	HVAC HVACServices
//	ExteriorLighting ExteriorLightingServices
//	InteriorLighting InteriorLightingServices
}

type SeatServices struct {
	GetPropertiesSeating func(VehicleHandle) GetPropertiesSeatingOutput
	MoveSeat func(VehicleHandle, MatrixId, string, Percentage, string, func(MoveSeatOutput)) MoveSeatOutput
}

type HVACServices struct {
	Service1 func(VehicleHandle) GeneralOutput
}*/

type ConnectivityData struct {
	PortNo string
	Protocol string
}

type ActiveService struct {
	serviceId uint32
	messageId string
	callback interface{}
	next *ActiveService
}

type ConnectedData struct {
	protocol string
	socket string
	clientTopic string
	connHandle interface{}  //*websocket.Conn, *grpc....
	responseChan chan map[string]interface{}
	activeService *ActiveService
	next *ConnectedData
}

type VehicleConnection struct {
	vehicleGuid string
	vehicleId VehicleHandle
	ipAddress string
	connectivityData []ConnectivityData
	connectedProtocol string
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

type GeneralOutput struct {
	Status ProcedureStatus
	Error *ErrorData
}

func generateRandomUint32() VehicleHandle {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return (VehicleHandle)(r.Uint32())
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
				if (*vehConn).connectedProtocol == protocol {
					(*vehConn).connectedProtocol = ""
				}
				*iterator =(*iterator).next
				break
			}
			iterator = &(*iterator).next
		}
	}
}

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
}

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
}

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

func removeActiveService(connectedDataList **ConnectedData, protocol string, requestId string) {
	if *connectedDataList == nil {
		fmt.Printf("removeActiveService: connectedDataList is empty for protocol=%s, requestId=%s\n", protocol, requestId) // should not be possible...
		return
	} else {
		iterator := connectedDataList
		for *iterator != nil {
			if (*iterator).protocol == protocol {
				activeServiceIterator := &(*iterator).activeService
				for *activeServiceIterator != nil {
					if (*activeServiceIterator).messageId == requestId {
						*activeServiceIterator = nil
						return
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
			}
			iterator = &(*iterator).next
		}
	}
}

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
					if (*activeServiceIterator).messageId == requestId {
						(*activeServiceIterator).messageId = subscriptionId
//fmt.Printf("updateActiveServiceKey: updated key %s->%s\n", requestId, subscriptionId)
						return
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
			}
			iterator = &(*iterator).next
		}
	}
}

func getSubscriptionId(connectedDataList *ConnectedData, protocol string, serviceId uint32) string {
	if connectedDataList == nil {
		fmt.Printf("getSubscriptionId: connectedDataList is empty for protocol=%s, serviceId=%d\n", protocol, serviceId) // should not be possible...
		return ""
	} else {
		iterator := connectedDataList
		for iterator != nil {
			if (*iterator).protocol == protocol {
				activeServiceIterator := (*iterator).activeService
				for activeServiceIterator != nil {
					if (*activeServiceIterator).serviceId == serviceId {
//fmt.Printf("getSubscriptionId: subscriptionId = %s\n", (*activeServiceIterator).messageId)
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

func saveReturnHandle(connectedDataList **ConnectedData, protocol string, requestId string, responseChan chan map[string]interface{}, callback interface{}) uint32 {
	if *connectedDataList == nil {
		return math.MaxUint32
	} else {
		iterator := connectedDataList
		var serviceId uint32 = math.MaxUint32
		if callback != nil {
			serviceId = uint32(generateRandomUint32())
			if serviceId == math.MaxUint32 {
				serviceId--
			} else if serviceId == 0 {
				serviceId++
			}
		}
		for *iterator != nil {
			if (*iterator).protocol == protocol {
				(*iterator).responseChan = responseChan
				activeServiceIterator := &(*iterator).activeService
				if *activeServiceIterator == nil {
					var activeService ActiveService
					activeService.messageId = requestId
					activeService.serviceId = serviceId
					activeService.callback = callback
					*activeServiceIterator = &activeService
					return serviceId
				}
				for *activeServiceIterator != nil {
					if (*activeServiceIterator).next == nil {
						var activeService ActiveService
						activeService.messageId = requestId
						activeService.serviceId = serviceId
						activeService.callback = callback
						(*activeServiceIterator).next = &activeService
						return serviceId
					}
					activeServiceIterator = &(*activeServiceIterator).next
				}
			}
			iterator = &(*iterator).next
		}
	}
	return 0
}

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

func sendMessage(vehicle *VehicleConnection, clientMessage string) {
	switch vehicle.connectedProtocol {
		case "VISSv3.0-wss": fallthrough
		case "VISSv3.0-ws":
			conn := getConnHandle(vehicle.connectedData, "VISSv3.0-ws").(*websocket.Conn)
			sendMessageWs(vehicle, conn, clientMessage)
		case "grpc":
		case "mqtt":
		case "http":
//		default: response =  `{"error": {"number": "502", "reason": "bad_gateway", "description": "The active protocol is not supported."}}`
	}
}

func sendMessageWs(vehConn *VehicleConnection, conn *websocket.Conn, clientMessage string) {
	err := conn.WriteMessage(websocket.BinaryMessage, []byte(clientMessage))
	if err != nil {
		fmt.Printf("Request error:%s\n", err)
	}
}

func receiveMessageWs(vehicleId VehicleHandle, conn *websocket.Conn, eventChan chan map[string]interface{}) {
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
		protocol := vehConn.connectedProtocol
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
}

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
	if vehConnList == nil {
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
	}
	return nil
}

func reformatOutput(messageMap map[string]interface{}, outputType string) interface{} {
	switch outputType {
		case "get":
			return reformatGetMessage(messageMap)
		case "subscribe":
			return reformatSubscribeMessage(messageMap)
		case "unsubscribe":
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
fmt.Printf("Socket=%s\n", socket)
	if strings.Contains(protocol, "ws") {
		return initVissV2WebSocket(socket)  // TODO: switch on protocol
	} else if strings.Contains(protocol, "grpc") {
		return nil //not yet implemented
	}
	return nil
}

// ****************** Common services ***************
func GetVehicle(vehicleGuid string) GetVehicleOutput {
	var vehConn VehicleConnection
	var out GetVehicleOutput
	vehConn.vehicleGuid = vehicleGuid
	vehConn.connectivityData, vehConn.ipAddress = getSupportedConnectivity(vehicleGuid)
	if vehConn.ipAddress == "" {
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "unknown vehicle")
		return out
	}
	vehConn.vehicleId = generateRandomUint32()
	addVehicleConnection(&vehConn)
	out.VehicleId = vehConn.vehicleId
	out.Protocol = make([]string, len(vehConn.connectivityData))
	for i := 0; i< len(vehConn.connectivityData); i++ {
		out.Protocol[i] = vehConn.connectivityData[i].Protocol
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
	for i := 0; i < len(vehConn.connectivityData); i++ {
		if vehConn.connectivityData[i].Protocol == protocol {
			matchingIndex = i
			break
		}
	}
	if matchingIndex >= 0 {
		var connectedData ConnectedData
		connectedData.protocol = protocol
		connectedData.socket = vehConn.ipAddress + ":" + vehConn.connectivityData[matchingIndex].PortNo
		if strings.Contains(protocol, "mqtt") || strings.Contains(protocol, "MQTT") {
			connectedData.clientTopic = generateRandomString()  //needed for VISSv3.0-mqtt
		}
		connectedData.connHandle = connectToVehicle(protocol, connectedData.socket)
		if connectedData.connHandle != nil {
			addConnectedData(&(vehConn.connectedData), &connectedData)
			vehConn.connectedProtocol = protocol
			go receiveMessageWs(vehicleId, connectedData.connHandle.(*websocket.Conn), eventChan) //alla protokollen ska ha samma eventChan
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
		out.Error = getErrorObject(400, "invalid_data", "Protocol not connected")
		out.Status = FAILED
		return out
	}
	// TODO: gör unsubscribe på activa services??
	switch protocol {
		case "VISSv3.0-wss": fallthrough
		case "VISSv3.0-ws":
			getConnHandle(vehConn.connectedData, protocol).(*websocket.Conn).Close()
		default:
			fmt.Printf("Disconnect: protocol not supported\n")
			out.Error = getErrorObject(400, "invalid_data", "Protocol not supported")
			out.Status = FAILED
			return out
	}
	responseChan := getResponseChan(vehConn.connectedData, protocol)
	m := make(map[string]interface{})
	m["error"] = "terminate"   // not read by receiveMessageXX anyway ...
	responseChan <- m
	removeConnection(&vehConn, protocol)
	out.Status = SUCCESSFUL
	return out
}

func SelectProtocol(vehicleId VehicleHandle, protocol string) GeneralOutput {
	var out GeneralOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn != nil {
		for i := 0; i<len(vehConn.connectivityData); i++ {
			if vehConn.connectivityData[i].Protocol == protocol {
				if getConnHandle(vehConn.connectedData, protocol) != nil {
					vehConn.connectedProtocol = protocol
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

func ServiceInquiry(vehicleId VehicleHandle) ServiceInquiryOutput {
// TODO: getting the relevant metadata from the vehicle about supported services
	var out ServiceInquiryOutput
	out.Status = FAILED
	out.Error = getErrorObject(503, "service_unavailable", "Service not implemented")
	return out
}

// ****************** Signal services ***************
func Get(vehicleId VehicleHandle, path string, filter string, stCredentials string) GetOutput {
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		var out GetOutput
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
	clientMessage := `{"action":"get", "path":"` + path + "\"" + filterParam + stCredParam + `, "requestId":"` + requestId + `"}`
	responseChan := make(chan map[string]interface{})
	serviceId := saveReturnHandle(&vehConn.connectedData, vehConn.connectedProtocol, requestId, responseChan, nil)
	if serviceId == math.MaxUint32 {
		var out GetOutput
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	sendMessage(vehConn, clientMessage)
	var responseMap map[string]interface{}
	select {
		case responseMap = <- responseChan:  //wait for response from receiveMessage
		
	}
	return reformatOutput(responseMap, "get").(GetOutput)
}

func Subscribe(vehicleId VehicleHandle, path string, filter string, stCredentials string, callback func(SubscribeOutput)) SubscribeOutput {
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		var out SubscribeOutput
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
	clientMessage := `{"action":"subscribe", "path":"` + path + "\"" + filterParam + stCredParam + `, "requestId":"` + requestId + `"}`
	responseChan := make(chan map[string]interface{})
	serviceId := saveReturnHandle(&vehConn.connectedData, vehConn.connectedProtocol, requestId, responseChan, callback)
	sendMessage(vehConn, clientMessage)
	var responseMap map[string]interface{}
	select {
		case responseMap = <- responseChan:  //wait for response from receiveMessage
		
	}
	responseMap["serviceId"] = serviceId
	return reformatOutput(responseMap, "subscribe").(SubscribeOutput)
}

func Unsubscribe(vehicleId VehicleHandle, serviceId uint32) GeneralOutput {
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		var out GeneralOutput
		out.Status = FAILED
		out.Error = getErrorObject(400, "invalid_data", "Vehicle is not connected")
		return out
	}
	subscriptionId := getSubscriptionId(vehConn.connectedData, vehConn.connectedProtocol, serviceId)
	requestId := generateRandomString()
	clientMessage := `{"action":"unsubscribe", "subscriptionId":"` + subscriptionId + `", "requestId":"` + requestId + `"}`
	responseChan := make(chan map[string]interface{})
	saveReturnHandle(&vehConn.connectedData, vehConn.connectedProtocol, requestId, responseChan, nil)
	sendMessage(vehConn, clientMessage)
	var responseMap map[string]interface{}
	select {
		case responseMap = <- responseChan:  //wait for response from receiveMessage
		
	}
	//  getConnHandle(vehConn.connectedData, protocol).(*websocket.Conn).Close() ??? jmf Disconnect
	return reformatOutput(responseMap, "unsubscribe").(GeneralOutput)
}

// ****************** Seat services ***************
type DirectionType uint8
const (
	Longitudal = iota  //Forward-backward direction of the vehicle; 0=forward-most, 100=backward-most
	Lateral            // Left-right direction of the vehicle; 0=left-most, 100=right-most
	Vertical           // Up-down direction of the vehicle; 0=down-most, 100=up-most
	Tilt               // Seat forward-backward tilt; 0=forward-tilt-most, 100=backward-tilt-most
	Lumbar             // Seat increase-decrease lumbar; 0=decrease-most, 100=increase-most
)

//Asynchronous move is realized by setting the position to DIRECTION_MAX or DIRECTION_MIN and then cancel the service asynchronously
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
	ServiceId uint32
}

type MoveSeatOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Position Percentage
}

type RowDef struct {
	RowName string
	ColumnName []string
}

type RaggedMatrix []RowDef

type GetPropertiesSeatingOutput struct {
	Status ProcedureStatus
	Error *ErrorData
	Id RaggedMatrix
	Movement []SeatMovementType
}

type SeatMovementType struct {
	Name string
	Description string
}

func MoveSeat(vehicleId VehicleHandle, seatId MatrixId, movementType string, position Percentage, stCredentials string, callback func(MoveSeatOutput)) MoveSeatOutput {
	var out MoveSeatOutput
	out.Status = SUCCESSFUL
	out.Position = 50
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
	if len(vehConn.connectedProtocol) == 0 {
		out.Error = getErrorObject(400, "invalid_data", "vehicle not connected")
		out.Status = FAILED
		return out
	}
	out.Status = SUCCESSFUL
	out.Id = []RowDef{{"Row1", []string{"Left", "Right"}}, {"Row2", []string{"Couch"}}}
	out.Movement = []SeatMovementType{{"longitudinal", "Seat movement in the direction parallel to the driving direction"},
	{"lateral", "Seat movement in the direction anti-parallel to the driving direction, in the horizontal plane"}}
	return out
}

// HVAC services
func hvacService1(vehicleId VehicleHandle) GeneralOutput {
	var out GeneralOutput
	vehConn := getVehicleConnection(vehicleId)
	if vehConn == nil {
		out.Error = getErrorObject(400, "invalid_data", "Protocol not connected")
		out.Status = FAILED
		return out
	}
	out.Status = SUCCESSFUL
	fmt.Printf("hvacService1:succefully called")
	return out
}
