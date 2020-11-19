# Introduction
What are the biggest problems in IoT? Where do current market IoT solutions fall short?  How should IoT be considered?

Put another way, _"What are the most common issues preventing companies from fully realizing the benefits of IoT?"_  This question was answered at [DZone](https://dzone.com/articles/most-common-problems-with-iot) by 23 executives involved with the Internet of Things.

**Observations:**
- Companies are not able, or do not have the talent, to complete the end-to-end solution.
- The unrealized complexity of deployment and the lack of skills to do so.
- Lack of seamless and secure data fabric platform.
- Challenging to make something at scale while maintaining quality.
- Creating scalable devices that connect to everything they need to.
- Large amount of the data that will run the IoT will be stored in the cloud.


The IoT industry, across numerous market verticals, is at an impasse where customers are demanding increasing sophistication at lower prices. Given the complexity and importance of IoT, no single company can or should be dictating the path forward for the entire industry.

The only viable path forward is collaboration between companies and market verticals to collaborate on developing, testing and standardizing the non-differentiating functionality. The [Open Connectivity Foundation](https://openconnectivity.org/) is currently working to achieve that, however the device-cloud communication represents a unique challenge for the engineers involved because there has never been a historical need for engineers to become knowledgeable in both embedded systems and cloud native application development. The proposed solution to this problem is to emulate the container runtime interface (CRI) architecture and embody Conway’s law to establish a loose coupling between the "IoT code" (CoAP/IoTivity cloud interface) and the portions of the system that are much more familiar to the cloud developers (ex: db/messaging/auth) which will also vary more depending on the use case.

## Challenges
- Embedded systems engineering and cloud native application development are likely orthogonal skill sets for the organizations whose products would benefit the most from internet connectivity
- The immense complexity of managing your own deployment means the market requires managed services
- There is no seamless portability for IoT devices between clouds
    - Extremely important if we want to decouple the networking costs from hardware costs for customers, like we do for cell phones
- Lack of an industry standard IoT cloud increases the attack surface of the industry

## plgd Goals

- Address these challenges in a way that is easy for companies and public clouds to adopt and offer as a managed service
- Ensure a loose coupling between the database / messaging / auth and the plgd Cloud implementation
- Run a system that demonstrates how to integrate the database / messaging / auth and serve as the default choice for companies with common OLTP use cases
