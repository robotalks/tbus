# Things Bus

**WORK IN PROGRESS**

_SHORT_: A Compact RPC (Remote Procedure Call) Protocol

_LONG_: A master driven software Bus protocol inspired by PCI/USB bus, based on
Google [protobuf](https://github.com/google/protobuf). By standardizing the
interface of commonly used peripheral devices, the controlling logic is modulized
and re-usable.

## Idea

A _bus_ is a logical group of peripheral devices.
A _master_ connects to the _bus_ and runs controlling logic.
A _bus_ itself is also a device, so it can be attached to another _bus_, thus
building a complex hierarchical bus tree is possible.
The communication between master and bus, bus and device can be remote.
The transportation is abstract, and the protocol is specially designed to make
communication over serial port possible.

### Motivation - why?

When building a robot, a lot of peripherals are common, e.g. sensors, servos,
step motors, DC motors etc.
The interface of these devices can be standardized, and the robot itself becomes
a collection of these devices.
On the controlling side, many logics are re-usable, e.g. controlling left/right
motors for different motion behaviors.
With standardized device interfaces, the controlling logics can be easily wrapped
in highly re-usable modules.

### Goals

- Small footprint: the protocol should be able to run on micro controllers
- Error proof: the protocol must have recovery mechanism if transmission error encountered

## Solution

[Protobuf](https://github.com/google/protobuf) is a very efficient and compact
binary message encoding format, and fits our goals best.
The only thing left is to build an RPC protocol on top of it.
See [Protocol](docs/specs/Protocol.md) for a draft of the RPC wire format.
And also in the folder `proto`, standardized commonly used device interfaces are defined.

## TODO

- Notification streams from device, for sensors
- Bus/Board matching
- Error recovery on serial transportation
- Support C/C++, especially for Arduino
- Support Java, for Android
- Support Python, preferred for AI
