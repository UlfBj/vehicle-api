# Seating Service Group
The Seating service group defines the services related to vehicle seats. This involves service like moving the seat forward/rearwar/up/down, activating massage, ventilation, retrieving seating configuration data, etc.

## Seating Datatypes
_*SeatLocationLayout:*_

```
typedef string SeatLocationLayout
```
* The SeatLocationLayout datatype is a string defining a JSON object that for each seating row contains a key-value pair where the key is the name of the row
and the key is an array containing the names of the seat locations of that row.
* SeatLocationLayout example:
   * {"Row1": ["DriverSide", "PassengerSide"], "Row2": ["DriverSide", "Middle", "PassengerSide"]}

---

_*SeatLocation:*_

```
struct SeatLocation {
    string SeatRow
    string SeatRowLoc
}
```
* SeatRow represents the name of the seat row.
* SeatRowLoc represents the name of the seat location on the row that SeatRow addresses.

The SeatLocation contains the name of a seating row and a seat location on that row, thus uniquely addressing one specific seat.
The values of SeatRow and SeatRowLoc must be present in the same key-value pair of the SeatLocationLayout for the vehicle.

## Seating Properties
The following procedures enable a caller to retrieve seating configuration data.

### Seat location layout
_*Signature:*_
```
SeatLocationLayout Seat_Property_LocationLayout(int32 vehicleHandle)
```
_*Service description:*_

The Seat_Property_Location service returns the vehicle seating layout configuration. 
*Input parameters:*

vehicleHandle:

The vehicle handle uniquely identifies a specific vehicle. See [GetVehicleHandle](/VSAPI/Specification/README.md#GetVehicleHandle).

*Output parameter:*

The output parameter has the datatype SeatLocationLayout which is a struct that has the format defined in the Seating Datatypes chapter.

---

## Seating Services
The following procedures define the services in the Seating service group.

### xxxx

