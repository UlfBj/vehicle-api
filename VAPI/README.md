# VSAPI Overview
The VSAPI is a technology agnostic API that defines Services structured in Service Groups. The services are used to invoke Service requests to change the vehicle state, to get information about the current vehicle state, or to obtain vehicle configuration data.
The Services are structured in Service Groups such as Seat, HVAC, etc. A service is defined by its procedure signature that is defined in a langue agnostic format that supports simple translation into most programming languages, see VSAPI Common Rules below.

The implementation of the VSAPI procedures can be done using different programming languages, and different implementations of the vehicle interface that is used for serialization and transport of the messages between the calling client and the server actuating the service. 

The VSAPI procedure typically executes in the context of the caller, while the server actuating the service typically executes in a different process, and likely on a different execution platform, se the figure below.

The figure below shows an architecture where the service endpoint is deployed on the vehicle.
![VSAPI architecture with remote service endpoint](/images/VSAPI-architecture-remote-endpoint.jpg)
Service endpoint implementations that are deployed at the vehicle are typically OEM proprietary.
Common and public service endpoint implementations can alternatively be deployed at the client side in an architecture as shown in the figure below.
![VSAPI architecture with remote service endpoint](/images/VSAPI-architecture-local-endpoint.jpg)

## VSAPI Structure
The VSAPI specification is found in the Specification directory structure. The general rule set, common data structures and procedures,
etc. are defined in the top level directory, while the  specifications that are specific for each Service group are defined in separate directories.

The language specific implementations are in separate directories under the top level Implementations directory.
The language implementations include one or more transport interface implementation.

## VSAPI current state
The development of the VSAPI shall be built on the needs of the automotive industry,
thus it needs to get input from organisations within the automotive domain to be able to create solutions meeting common needs of the domain.
The initial VSAPI development was done on input from [COVESA](https://covesa.global/) where an analysis was done for the Seating capabilities,
so this Service group is therefore the first to be specified and implemented.
The list below will over time grow from this initial state.

### VSAPI Service Group specifications
* Seating

### VSAPI Service Group Language Implementations
* Go

### VSAPI Service Group Transport Implementations
* VISSR / VSAPI-HIM
