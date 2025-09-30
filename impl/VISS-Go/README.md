# VAPI Go Implementation
Different VAPI implementations in Go can be found here.
The implementations mainly differ on the interfaces that are used in the communication with the vehicle servers.
The list below shows the different implementations.
* Vehicle communication based on VISSv3.0 

When testing this VAPI implementation it requires a VISS server that it can connect to.
This can be achieved by running an instance of the VISSR server which can be done by the following steps.
1. Clone the [VISSR)(https://github.com/COVESA/vissr) repo.
2. In the VISSR root directory, initiate the command: $ ./runstack.sh startme

then build vapiTest.go

$ go build -o vapiTest

and run vapiTest that connects to the VISSR server on localhost.

$ ./vapiTest

# VSS massage exensions
The service ActivateMassage requires the following nodes to be added to the standard VSS tree.
They should be added to the Cabin/Seat.vspec file, below the 'Switch.Massage' branch definition
```
# ----- VAPI additions -----
Switch.Massage.IsOn:
  datatype: boolean
  description: Indicates if massage is on or off. True = On. False = Off.
  type: actuator

Switch.Massage.Intensity:
  datatype: float
  description: The intensity of the massage that will be activated. Min intensity = 0. Max intensity = 100.
  type: actuator

Switch.Massage.MassageType:
  datatype: string
  description: The type of massage that will be activated.
  type: actuator
```

# Service invokation achitecture
The image below shows the architecture for the message flows when a client invokes a service.
![VAPI service invokation architecture](/images/vapi-service-invokation-arch.jpg)
The readMessage thread is spawned by the call to the Connect service and is shared by all services that uses the connecion/protocol setup by that Connect call.
A service starts with spawning itself on a thread then it provides the readMessage thread a reference to a back channel over which all messages read from the vehicle server are returned by saving the channel reference in a commonly shared data structure.
When the service either ends successfully or due to an error, the service thread is terminated.
The readMessage thread is not terminated until the Disconnect service is called to terminate the connection for the protocol it handles.

The shared data structure is updated accordingly when services are initiated/terminated, and when connections to a vehicle server are initiated/terminated.
