# VAPI Specification Overview
VAPI defines procedures with input and output parameters for the access of both signals and services.
For signal access there are the following set of reserved methods available:
* Get
* Set
* Subscribe
* Unsubscribe
* GetMetadata

This set of procedures is used for access to any vehicle signals.

For services there is one procedure specified for each service,
defined by its procedure signature that is defined in a langue agnostic format that supports simple translation into most programming languages.
The Services are structured in Service Groups such as Seat, HVAC, etc.



The VAPI is a technology agnostic API that defines Services structured in Service Groups. The services are used to invoke Service requests to change the vehicle state, to get information about the current vehicle state, or to obtain vehicle configuration data.
The Services are structured in Service Groups such as Seating, HVAC, etc. A service is defined by its procedure signature that is defined in a langue agnostic format that supports simple translation into most programming languages, see VAPI Common Rules below.

# VAPI Rule Set
The definition of VAPI services shall follow the rules described below.
* A service shall be defined by its procedure signature following the format shown below.
```
<output-datatype> procedure-name(<input-datatype input-parameter-name>)
```
Where
   * ‘output-datatype’ declares the data type of the output parameter. There can be zero or one output parameter. A complex data type may be used that defines multiple output parameters as its members.
   * ‘procedure-name’ is the name of the procedure that is invoked to actuate the service.
   * ‘input-datatype input-parameter-name’ are the data type and name of an input parameter. There can be zero or more input parameters, separated by a comma.

* The datatypes used in the signatures can be any of the primitive datatypes such as intx/uintx (x=8/16/32/64), float, double, boolean, string, but also complex datatypes such as structs with members being any of the primitive or complex datatypes.
* The output parameter shall at least contain a status parameter of the datatype ServiceStatus_t.
* The datatype ServiceStatus_t shall be an enumeration with the following semantics:
   * NOT_STARTED = 1
   * ONGOING = 2
   * SUCCESS = 0
   * FAILURE = -1
* The ServiceStatus_t value FAILURE may be extended with other negative values.
   * The symbolic value shall be FAILURE_X where X shall refer to the error reason.
* The service definition shall include descriptions of the service as well as of its input and output parameters,
and any other information that may be relevant for using the service.
* All service procedures shall be synchronous, i. e. they shall return immediately.
* If the service execution has a temporal duration such that its termination will be later than the procedure return, then the output parameter shall contain a ‘session identifier’, a reference to the invoked execution of the service.
   * The datatype of the session identifier shall allow random generation of instances that have a small likelihood to be a duplicate of a previously generated instance within a vehicle ignition cycle. All procedures within a service group must use the same datatype.
* All service groups shall expose a procedure that a client can use to asynchronously terminate an ongoing service execution. The procedure shall have the signature:
   * ServiceStatus_t <service-group-name>_Terminate(<datatype> sessionId)
* The input to a service that returns a session identifier shall include a pointer to a procedure to which the service can issue callbacks. A client may set this pointer to the nil value in which case the service will not issue any callbacks.
* A service that returns a session identifier shall issue asynchronous callbacks during its execution. It must issue a final callback at its termination, whether the termination has been successful or failed and it may issue more callbacks. The callback must contain a ServiceStatus_t parameter and may contain other parameters.
* If procedures of a service group require input that is dependent on the configuration of the target vehicle, then the service group shall expose procedures that enables a client to retrieve the required configuration data. These procedures shall have the signature:
```
    <output-datatype> <service-group-name>_Property_X()
```
Where X describes the wanted type of configuration data, and ‘output-datatype’ shall contain the requested configuration data together with a ServiceStatus_t parameter.
* A service may include in the output parameter data that represents the vehicle signals that may be changed by the execution of the service. The format of this data shall be a string array where each string represents the path of the signal as defined by the [COVESA VSS](https://github.com/COVESA/vehicle_signal_specification) project.

## VAPI Common Procedures
The following procedures are not specific to any Service group.

### GetVehicleHandle
_*Signature:*_
```
GetVehicleHandleOutput GetVehicleHandle(string pseudoVIN)
```
_*Service description:*_

The GetVehicleHandle procedure xxxx

*Input parameters:*

pseudoVIN:

The VAPI client must obtain a pseudoVIN that must be a globally unique reference to a specific vehicle.
How the pseudoVIN is obtained is out-of-scope for the VAPI specification.

*Output parameter:*
```
struct GetVehicleHandleOutput {
    uint32 VehicleRef
    ServiceStatus_t Status
}
```
The output parameter contains a vehicle reference that the client must provide as input to most VAPI services to identify the vehicle that the service shall be applied upon.
This reference does only have to be unique within the scope of the client process.


---
### CancelService
_*Signature:*_
```
(*Vehicle) CancelService(sessionId uint32) ServiceStatus_t
```

## VAPI Private Procedures
The private procedures are not accessible by VAPI clients, only by other VAPI procedures.

## getVehicleLink
_*Signature:*_
```
string getVehicleLink(uint32 vehicleRef)
```

The procedures use it to retrieve the information needed to connect to the possibly remote vehicle,
which in most cases is a socket (URL and port number)

The implementation of this procedure is OEM specific

