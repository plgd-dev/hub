# Design

# collection

Event:
{
    version: uin64,
    type: string,
    data: bytes,
}

Document:
{
    id:                   aggregateID.version
    aggregateID:          string,
    groupID:              string,
    latestVersion:        uint64,
    documentClosed:       bool,
    events:               []Event,
}

{
    #metadata
    id:                   aggregateID.version
    aggregateID:          string,
    groupID:              string,
    latestVersion:        uint64,
    documentClosed:       bool,
    events:               []Event,

    #snapshot data:
    ResourceChanged  latest_resource_change = 3;
    map[string]bool create_pending_requests_count = 4;
    map[string]bool delete_pending_requests_count = 5;
    map[string]bool update_pending_requests_count = 6;
    map[string]bool retrieve_pending_requests_count = 7;
    EventMetadata event_metadata = 5;
}



LoadFromSnapshot:
    - deviceID:
        getSnapshotDocument:
            Find(groupID == deviceID && isLastSnapshot == true).Sort(latestVersion:-1):
                doc.documentClosed == false:
                    iterate over doc.events
                doc.documentClosed == true:
                    Find( aggregateID == doc.aggregateID && startVersion > doc.startVersion).Sort(latestVersion:):
                        iterate over doc.events +  iterate over aggregateDocs.events
    - aggregateID:
        Find( aggregateID == doc.aggregateID && isLastSnapshot == true).Sort(-1).Limit(1):
            doc.documentClosed == false:
                iterate over doc.events
            doc.documentClosed == true:
                    Find( aggregateID == doc.aggregateID && startVersion > doc.startVersion):
                        iterate over doc.events +  iterate over aggregateDocs.events

SaveSnapshot:
    - version == 0


Save:
    version == 0:
        insert{Document{
            id: aggregateID.0,
            groupID: groupId,
            startVersion: 0,
            latestVersion: 0,
            firstEventIsSnapshot: true,
            documentClosed: false,
            isLastSnapshot: true,
            events: []Events{
                {
                    version: version,
                    type: type,
                    data: data,
                }
            }
        }}
    version > 0:
        update{
            Find({
                aggregateID: aggregateID,
                documentClosed: false,
                latestVersion: version-1,
            }),
            Update({
                Set:
                    latestVersion: version
                Push:
                    events: event:
            })
        }
        switch (err) {
        case nil: return ok
        case NotFound:
            update{
                Find({
                    aggregateID: aggregateID,
                    documentClosed: false,
                    latestVersion: >= version,
                }),
                OnInsert({
                    Document{
                        id: aggregateID.version,
                        groupID: groupId,
                        startVersion: aggregateID.version,
                        latestVersion: aggregateID.version,
                        firstEventIsSnapshot: false,
                        documentClosed: false,
                        isLastSnapshot: false,
                        events: []Events{
                            {
                                version: version,
                                type: type,
                                data: data,
                            }
                    }
                }, withUpsert)
                if doc == found && insert not used:
                    occ
        case MaxDocSizeExceeded:
            update{
            Find({
                aggregateID: aggregateID,
                documentClosed: false,
                latestVersion: version-1,
            }),
            Update({
                Set:
                    documentClosed: true
            })
            switch (err) {
                case nil:
                    insert{Document{
                        id: aggregateID.version,
                        groupID: groupId,
                        startVersion: aggregateID.version,
                        latestVersion: aggregateID.version,
                        firstEventIsSnapshot: false,
                        documentClosed: false,
                        isLastSnapshot: false,
                        events: []Events{
                            {
                                version: version,
                                type: type,
                                data: data,
                            }
                        }
                    }}
                    switch (err) {
                        case nil: return ok
                        case Exist: concurrency exception
                    }
                case NotFound: concurrency exception
            }
        }
