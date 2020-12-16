# System Overview

## System Context Diagram
![L1](/img/diagrams/system-plgd.svg)

## plgd Context Diagram
![L2](/img/diagrams/container-plgd.svg)

## CoAP Gateway
The CoAP gateway acts act as a CoAP Client, communicating with IoT Devices - CoAP Servers following OCF specification. As the component diagram describes, responsibilities of the gateway are:
- handle and maintain TCP connections comming from devices
- forward [authentication and authorization requests #5.4.4](https://openconnectivity.org/specs/OCF_Device_To_Cloud_Services_Specification_v2.2.0.pdf) #5.4.4 to Authorization Service
- process device CRUDN operations which are by its nature forwarded to [Resource Aggregate](resource-aggregate) or [Resource Directory](resource-directory)

### Operational flow
Device after it connects to the CoAP Gateway over TCP needs to send an authorization request. If it is the first connection, device needs to register. As the plgd.cloud uses standardized OAuth2.0 flow, device needs to firstly receive a unique [authorization code](https://tools.ietf.org/html/rfc6749#section-4.1) from the Client(#component-client) and proceed with registration called Sign Up request. Goal of the Sign Up request is to exchange the authorization code for an access and refresh token, which is after successful validation by the OAuth2.0 Server returned back to the device. 

@startuml Sequence
hide footbox

participant D1 as "Device #1"
participant CGW as "CoAP Gateway"
participant AS as "Authorization Server"
participant RA as "Resource Aggregate"

D1 -> CGW ++: Sign Up
group OAuth2.0 Authorization Code Grant Flow
    CGW -> AS ++: Exchange authorization code for access token
    AS -> AS: Verify Authorization Code
    return Ok\n(Access Token, Refresh Token, ...)
end
CGW -> RA ++: Register device resource
return Registered
return Signed up\n(Access Token, Refresh Token, ...)

@enduml

Successful registration to the plgd.dev is followed by authentication request called Sign In. Device needs to Sign In right after it sucessfully established the connection to the CoAP Gateway, otherwise it won't be marked as online - it won't be available. Sign In requests authorizes the opened TCP connection by verifying the Access Token.

@startuml Sequence
hide footbox

participant D1 as "Device #1"
participant CGW as "CoAP Gateway"
participant RA as "Resource Aggregate"

D1 -> CGW ++: Sign In
CGW -> CGW: Validate Accses Token
CGW -> RA ++: Declare device as online
return
return Signed In

@enduml

As device capabilities is represented by set of resources, device needs to inform the plgd.cloud which resources are available remotely by publishing them. Information which is published is not the resource representation itself, but [Resource Information #6.1.3.2.2](https://openconnectivity.org/specs/OCF_Device_To_Cloud_Services_Specification_v2.2.0.pdf).


```json
{
   "di":"e61c3e6b-9c54-4b81-8ce5-f9039c1d04d9",
   "links":[
      {
         "anchor":"ocf://e61c3e6b-9c54-4b81-8ce5-f9039c1d04d9",
         "href":"/myLightSwitch",
         "rt":[
            "oic.r.switch.binary"
         ],
         "if":[
            "oic.if.a",
            "oic.if.baseline"
         ],
         "p":{
            "bm":3
         },
         "eps":[
            {
               "ep":"coaps://[fe80::b1d6]:1111",
               "pri":2
            },
            {
               "ep":"coaps://[fe80::b1d6]:1122"
            },
            {
               "ep":"coaps+tcp://[2001:db8:a::123]:2222",
               "pri":3
            }
         ]
      }
   ],
   "ttl":600476
}
```

Published resource information doesn't contains only link to the resource to be able to CRUDN particular capability, but also resource types which are used to filter only resources client is able to use. As an example, if you have an application which controls the light, this app will search for all lights you have at home - filter resources type `oic.r.switch.binary`, and not for temperature sensors, even if it was published by the same device. This resource publish command registers the resource aggregate in the plgd.cloud what makes it discoverable to all authorized clients.

The plgd.cloud is after successful resource publish starting observation of each resource. The CoAP Gateway is registered as a client interested in representation changes of each resource, what builds eventually consistend resource shadow inside of the plgd.cloud. Response to the resource observation request contains [current representation](https://tools.ietf.org/html/rfc7641#section-1.1). Additional responses called [notifications](https://tools.ietf.org/html/rfc7641#section-3.2) are send whenever the state of the resource changes.

@startuml Sequence
hide footbox

participant D1 as "Device #1"
participant CGW as "CoAP Gateway"
participant RA as "Resource Aggregate"

D1 -> CGW ++: Publish Resources
CGW -> RA ++: Publish Resources
return
CGW -> D1 ++: Observe published resource
return Resource representation
CGW -> RA ++: Update resource representation
return

@enduml

![L3](/img/diagrams/component-coapgateway.svg)

## gRPC Gateway
![L3](/img/diagrams/component-grpcgateway.svg)

## HTTP Gateway
![L3](/img/diagrams/component-httpgateway.svg)

## Resource Aggregate
![L3](/img/diagrams/component-resourceaggregate.svg)

## Resource Directory
![L3](/img/diagrams/component-resourcedirectory.svg)

## Device
![L3](/img/diagrams/component-device.svg)

## Client
![L3](/img/diagrams/component-clients.svg)