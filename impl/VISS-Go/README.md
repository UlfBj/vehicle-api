# VAPI Go Implementation
Different VAPI implementations in Go can be found here.
The implementations mainly differ on the interfaces that are used in the communication with the vehicle servers.
The list below shows the different implementations.
* Vehile communication based on VISSv3.0 

Build vapiTest.go

$ go build -o vapiTest

Run vapiTest

$ ./vapiTest

# Service invokation achitecture
The image below shows the architecture for the message flows when a client invokes a service.
![VAPI service invokation architecture](/images/vapi-service-invokation-arch.jpg)
Before the client invokes a service the following must 
1. The client must call xxxx

The above is executed once, typically at 
1. A client calls the service method.
2. 
