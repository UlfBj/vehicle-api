# VAPI Go Implementation
Different VAPI implementations in Go can be found here.
The implementations mainly differ on the interfaces that are used in the communication with the vehicle servers.
The list below shows the different implementations.
* Vehile communication based on VISSv3.0 

Build vapiTest.go

$ go build -o vapiTest

Run vapiTest

$ ./vapiTest

# VSS massage exensions
The service ActivateMassage requires the following nodes to be added to the standard VSS tree.
They should be added to the Cabin/Seat.vspec file, below the 'Switch.Massage' branch definition
```
# ----- VAPI additions -----
Switch.Massage.IsOn:
  datatype: boolean
  description: Indicates if mssage is on or off. True = On. False = Off.
  type: actuator

Switch.Massage.Intensity:
  datatype: float
  description: Indicates if mssage is on or off. Min intensity = 0. Max intensity = 100.
  type: actuator

Switch.Massage.MassageType:
  datatype: string
  description: The type of massage that will be activated.
  type: actuator
```

# Service invokation achitecture
The image below shows the architecture for the message flows when a client invokes a service.
![VAPI service invokation architecture](/images/vapi-service-invokation-arch.jpg)
In case the service does not utilize event/callback reporting the SwCs in the lower left of the figure do not take part in the service message flow.
The sequence is in the case of no event reporting the following:
1. The Client calls the Service with any input parameters as specified by VAPI for the service.
2. The Service serializes the data according to the active protocol to be used, and issues a request to the remote server.
The logic that implements the service may involve more than one request being sent to the vehicle server, but for simplicity of the example it is assumed it is only one request.
3. The vehicle server acts on the request, creates a response which is received by the ReceiveMessage SwC. 
4. ReceiveMessage forwards the response to the Service.
5. The Service reformats the response data into the output format as specified by VAPI and returns it to the client.

In case the service does utilize event/callback reporting the the following is appended to the sequence above.

6. The vehicle server issues an event message to ReceiveMessage.

7. ReceiveMessage forwards the message to the EventHandler.

8. The EventHandler reformats the event data into the VAPI output format for the service nd forwards it to the callbackInterceptor.

9. The callbackInterceptor checks the event data to decide whether it is the last event being received as part of the invoked service call.
If so it issues a request to the vehicle server to terminate the event reporting for this ongoing session.

10. The callbackInterceptor forwards the callback message to the Client.

This is the high level description of the information flow that does not mention details such that:

A. The ReceiveMessage SwC, which is protocol specific, is instantiated by a previous client call to Connect,
and it is terminated if a later client call is made to Disconnect.

B. The EvenHandler is instantiated by a prevoius client call to GetVehicle, and it is terminated if a later client call is made to ReleaseVehicle.

C. The callbackInterceptor is instantiated by the Service. A service may be implemented without instantiation of a callbackInterceptor.
This would be the case if a callbackInterceptor does not have access to sufficient information to make a decision to terminate the callback session
by calling CancelService.
The responsibility of this termination is then placed on the client. The Service dokumentation shall clearly state where this responsibility lies.
