# VSAPI Implementations
The VSAPI is a technology agnostic API that defines Services structured in Service Groups. The services are used to invoke Service requests to change the vehicle state, to get information about the current vehicle state, or to obtain vehicle configuration data.
The [VSAPI Specification](/Specification/) describes the procedures with input and output parameters that are invoked to execute the services.
The implementation of these procedures can be done in different programming languages, and also using different technology solutions for the serialization and transport of the service information.
The directory structure supports this by having sparate directories for implementations in different languages,
and each of these directories can contain multiple directories for different technology solutions in that language.
