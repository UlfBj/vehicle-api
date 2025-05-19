# vehicle-api (VAPI)
The VAPI is a technology agnostic API for accessing vehicle signals and services.

The datamodel used in VAPI is the [Hierarchical Information Model](https://covesa.github.io/hierarchical_information_model/) (HIM).

Vehicle signal sets that can be accessed using VAPI can for example be found at:
* [Vehicle Signal Specification]() (VSS)  // passenger car centric signal set
* [Commercial Vehicle Information Specifications](https://covesa.github.io/commercial-vehicle-information-specifications/) (CVIS)  // truck, trailer, bus etc centric signal sets

The signal sets referenced above are merely examples, VAPI is agnostic to what content a signal set has.

For services there do not yet exist any stable publicly accessible sets,
so the initial input to the definition of the VAPI services comes from the [COVESA CVI - Capabilities](https://wiki.covesa.global/display/WIK4/Capabilities+Project) project,
which e. g. contains a definition of the [Seating capabilities](https://wiki.covesa.global/display/WIK4/Capabilities+Project+-++Seating+Capabilities).

In VAPI a service is defined as a procedure with input and output parameters having a one-to-one mapping with corresponding data from a set of services defined using the HIM service profile.

The signal-oriented part of VAPI uses the following reserved procedure names:
* Get
* Set
* Subscribe
* Unsubscribe
* GetMetadata

For services there is one procedure specified for each service.

The procedure typically executes in the context of the caller, while the server actuating the service typically executes in a different process,
and likely on a different execution platform.
//  TODO: add pictures for signal arch and diff variants of service arch.

The protocol used for the communication between client and server is not specified by VAPI, it is an implementation decision.
The currently only existing implementation uses the COVESA [Vehicle Information Service Specification]() (VISSv3.x),
But VAPI is agnostic to which protocol that is used in the procedure implementations.
The existing implementation is written in Go. While a Go library can be integrated with C code implementations,
ideally library implementations in other languages will be added later.

The VAPI is currently in an incubation phase, particularly it needs to be extened with many more service group APIs,
more examples ini different languages, etc. before it becomes a complete Vehicle Service API.

## Contributors
VAPI is an open standard and we invite anybody to contribute. Currently VAPI contains - among others - significant  contributions from

 - [Ford Motor Company](https://www.ford.com/)
