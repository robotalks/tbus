# TBus Protocol Specification

## What is TBus?

TBus is Things Bus which is a software defined bus for Things.
The typical use of TBus is to connect Robots and peripherals including
motors, servos, sensors etc. Like PCI, USB buses, it has the capability of
enumerating the devices on the bus, transferring messages.

For example, using a smart phone to control a robot via Wifi. The App on the
phone connects to the bus (a HTTP or TCP connection over Wifi) as master, and
the Robot also connects to the bus as a bus enumerator, which enumerates
the peripherals (motors, servos, cameras, gyro, accelerometer, etc.), and
the App is able to directly control the peripherals by sending control messages.

In a larger scenario, a service on Internet connects to multiple Robots remotely.
The service itself behaves as a bus enumerator which enumerates connected Robots
as bus enumerators. When the client of the service connects to the bus, it's able
to discover the connected Robots, and control them directly.

Further more, a smart phone (or even a PC) can also be a bus enumerator which
enumerates microphone, speaker, compass, accelerometer, camera, etc.

## Device Addressing and Identification

Each device on the bus is addressed using a bus local address. To send message to
a device connected on a remote bus, the message is encapsulated in a forward message
which addresses the bus enumerator.

Regarding the functionality of a device, like USB device, a class ID and a device
ID are used. The class ID defines the common functionalities in a set of methods
with related data contracts (aka interface in some programming languages). The
device ID defines the device specific functionalities which may support extended
methods.

The message sent from master to device is remote method invocation, and the
message from device master is the result of invocation.

## Transportation

The protocol can work on any 8-bit byte-stream transportation, like serial port,
TCP protocol, WebSocket etc. The transportation is not necessary to be reliable.
Take serial port as example, transfer error is possible. The protocol doesn't
support re-transmission. On application level, it may report error or timeout.

For unreliable transportation like serial port, the transportation layer can optionally
wrap the message with prefix and suffix.

### Message Prefix and Suffix

#### Transmission Error Detection

These prefix/suffix bytes can be used by serial transportation for error detection.

Prefix = 0xa5
Suffix = CheckSum[0:7] CheckSum[8:15] 0x5a

#### Routing

Prefix[7..5] = 0b110
Prefix[4..0] = NumberOf(Addresses)-1

When sent Master-to-Device, it's a routing request, when sent Device-to-Master,
it's routing error.

## Message Format

### Packet

Field         | Bytes | Content
--------------|-------|--------
Flags         | 1     | message flags
MsgIDSize     | 1-4   | 7-bit encoded message id size
BodySize      | 1-4   | 7-bit encoded body size in bytes
[MsgID]
[Body]

### Flags

Bit | Field     | Content
----|-----------|--------
4-7 | Format    | 0001 - rev1, ProtoBuf encoded
1-3 | Reserved  | 0
0   | Event     | 1 indicate this is an event from device to master

### Body - Master to device

Field  | Bytes | Content
-------|-------|--------
Method | 1     | bit[7]: reserved, bit[0-6]: method index
Params | n     | method parameters

### Body - Device to master

Field    | Bytes | Content
---------|-------|--------
RepFlags | 1     | reply flags
Result   | n     | result

### RepFlags

Bit | Field    | Content
----|----------|--------
7   | Error    | 0 - normal, 1 - error and result is encoded error
0-6 | Reserved | 0

## Device Classes

Each device class has pre-defined list of methods and parameters/responses.
The classes are defined using ProtoBuf services.
