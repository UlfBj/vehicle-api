# Vehicle API (VAPI)
The VAPI Github repo contains both the VAPI specification and implementations of the VAPI specification.
While the VAPI specification is connectivity agnostic, implementations will be restricted to one (or more) connectivity solutions.
The term connectivity solution refer in this context to the connectivity solution that is used between client and server,
more specifically the message serialization/de-serialization, and the transport protocol for the message communication.

A rendered version of the current state of the VAPIv1.0 specification is found on this link

[VAPIv1.0](https://raw.githack.com/UlfBj/vehicle-api/main/spec/VAPIv1.0.html)

Please note that this is a specification under development, there might be substantial changes to it before it is released.

The VAPI specification is found in the spec directory.
VAPI implementations are found in the impl directory, where implemenations for different connectivity solutions and languages can be found.
The following connectivity/language combinations are available:
* [VISSR (VISSv3.x)](https://github.com/COVESA/vissr) / Go

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

For a client using the signal support in VAPI the logical architecture would typically look as shown below.
![VAPI architecture with remote signal endpoints](/images/vapi-architecture-base-signals.jpg)
While signal endpoints are always on the "remote" side of the server,
for services their implementations can be deployed either remotely or locally, as shown in the figures below.
![VAPI architecture with remote service endpoints](/images/vapi-architecture-remote-endpoint.jpg)
![VAPI architecture with local service endpoints](/images/vapi-architecture-local-endpoint.jpg)

The protocol used for the communication between client and server is not specified by VAPI, it is an implementation decision.
The currently only existing implementation uses the COVESA [Vehicle Information Service Specification]() (VISSv3.x),
but VAPI is agnostic to which protocol that is used in the procedure implementations.
The existing implementation is written in Go. While a Go library can be integrated with C code implementations,
ideally library implementations in other languages will be added later.

The VAPI is currently in an incubation phase, particularly it needs to be extened with many more service group APIs,
more examples in different languages, etc. before it becomes a comprehensive Vehicle API.

A rendered version of the VAPIv1.0 specification is found on this link:

https://raw.githack.com/UlfBj/vehicle-api/main/spec/VAPIv1.0.html

## Contributors
VAPI is an open standard and we invite anybody to contribute. Currently VAPI contains - among others - significant  contributions from

 - [Ford Motor Company](https://www.ford.com/)
