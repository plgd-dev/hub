# Event Store
plgd cloud persist events in an event store, which is a database of events. The store has an API for adding and retrieving device's / resource's events. Events needs to be versioned and written in a correct order. To achieve the consistency, optimistic concurrency control method is applied during each write.
After the event is successfuly written into the event store, event is distributed to the event bus to all interested parties.

plgd Cloud defines EventStore interface what allows integration of different technologies to store the events. During the last 2 years the project evaluated multiple technologies, e.g.
- CockroachDB
- Apache Kafka
- MongoDB
- NATS Jetstream
- Google Firestore

Currently supported and preffered solution is MongoDB. Details how this integration works can be found below.

## MongoDB
Device's data are in the MongoDB organized per devices. For each connected device a new collection is created. Each event is modeled as a new document.


### Queries
#### Query resources B of device d9dd7...
1. Get Latest Snapshot
    a. Find document where _id == B.s
    b. Get version of latest snapshot event
2. Find documents where aggregateID == B && version >= latestSnapshotVersion

#### Query all resources of device d9dd7...
1. Get Latest Snapshots of all resources
    a. Find documents where aggregateID == snapshot && version == -1
    b. Get versions of latest snapshot events per each resource
2. Find all documents per each resource where aggregateID == snapshot.aggregateID && version >= latestSnapshotVersion