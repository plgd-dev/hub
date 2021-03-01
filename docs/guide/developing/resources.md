# Working with Resources
## Creating Resources

### Description

Device with collection resource allows dynamic creation of resources. The created resource can only be of a well defined type (see call to `oc_collections_add_rt_factory` in [Guide](#define-constructor-and-destructor)) and all created resources are contained within the collection.

### Guide

To develop your own device you can examine the example in [cloud_server](https://github.com/iotivity/iotivity-lite/blob/master/apps/cloud_server.c). Lets examine the example to identify the necessary steps that allow a device to dynamically create resources on a collection.

#### Create a collection resource
```C
oc_resource_t* col = oc_new_collection(NULL, "/switches", 1, 0);
oc_resource_bind_resource_type(col, "oic.wk.col");
```
For precise description of arguments of given functions please refer to the iotivity-lite documentation.

#### Determine which resource types can populate the collection
```C
oc_collection_add_supported_rt(col, "oic.r.switch.binary");
```

#### Enable create operation on the collection resource
```C
oc_resource_bind_resource_interface(col, OC_IF_CREATE);
```

#### Define constructor and destructor
```C
oc_collections_add_rt_factory("oic.r.switch.binary", new_switch_instance, free_switch_instance);
```
```C
typedef struct oc_switch_t
{
  struct oc_switch_t *next;
  oc_resource_t *resource;
  bool state;
} oc_switch_t;
OC_MEMB(switch_s, oc_switch_t, 1);
OC_LIST(switches);

static oc_resource_t*
new_switch_instance(const char* href, oc_string_array_t *types,
                    oc_resource_properties_t bm, oc_interface_mask_t iface_mask,
                    size_t device)
{
  oc_switch_t *cswitch = (oc_switch_t *)oc_memb_alloc(&switch_s);
  if (cswitch) {
    cswitch->resource = oc_new_resource(
      NULL, href, oc_string_array_get_allocated_size(*types), device);
    if (cswitch->resource) {
      size_t i;
      for (i = 0; i < oc_string_array_get_allocated_size(*types); i++) {
        const char *rt = oc_string_array_get_item(*types, i);
        oc_resource_bind_resource_type(cswitch->resource, rt);
      }
      oc_resource_bind_resource_interface(cswitch->resource, iface_mask);
      cswitch->resource->properties = bm;
      oc_resource_set_default_interface(cswitch->resource, OC_IF_A);
      oc_resource_set_request_handler(cswitch->resource, OC_GET, get_cswitch,
                                      cswitch);
      oc_resource_set_request_handler(cswitch->resource, OC_POST, post_cswitch,
                                      cswitch);
      oc_resource_set_properties_cbs(cswitch->resource, get_switch_properties,
                                     cswitch, set_switch_properties, cswitch);
      oc_add_resource(cswitch->resource);
      oc_set_delayed_callback(cswitch->resource, register_to_cloud, 0);

      oc_list_insert(switches, prev, cswitch);
      return cswitch->resource;
    }
    oc_memb_free(&switch_s, cswitch);
  }
  return NULL;
}

static void
free_switch_instance(oc_resource_t *resource)
{
  oc_switch_t *cswitch = (oc_switch_t *)oc_list_head(switches);
  while (cswitch) {
    if (cswitch->resource == resource) {
      oc_delete_resource(resource);
      oc_list_remove(switches, cswitch);
      oc_memb_free(&switch_s, cswitch);
      return;
    }
    cswitch = oc_list_item_next(cswitch);
  }
}
```

#### Compile and link

To enable create operation in iotivity-light library compile with CREATE=1 option.

```make
make cloud_server CLOUD=1 SECURE=0 CREATE=1
```

#### Create a resource

When you have a cloud backend and cloud_server binary running. You can use cloud client to create a resource.

#### Using go grpc client

Go [grpc client](https://github.com/plgd-dev/cloud/tree/master/bundle/client/grpc) is a simple tool that supports several useful commands we can combine to create a resource.

1. Use the get command to identify the collection device

The get command retrieves data of all available devices. To correctly call the create command the device id and href properties are necessary. In the output find the item with type "oic.wk.col".

```bash
// retrieves all resources of all devices
./grpc -get
```
Output:
```json
...
{
    "content": {
        "content_type": "application/vnd.ocf+cbor",
        "data": "n/8="
    },
    "resource_id": {
        "device_id": "2b9ed3ed-ddf3-4c9c-4d21-9ec1f6ba6b03",
        "href": "/switches"
    },
    "status": 1,
    "types": [
        "oic.wk.col"
    ]
}
...
```
2. Create a binary switch resource in the collection
```bash
./grpc -create -deviceid 2b9ed3ed-ddf3-4c9c-4d21-9ec1f6ba6b03 -href /switches <<EOF
{
    "rt": [
        "oic.r.switch.binary"
    ],
    "if": [
        "oic.if.a",
        "oic.if.baseline"
    ],
    "rep": {
        "value": false
    },
    "p": {
        "bm": 3
    }
}
EOF
```

The command creates a binary switch ("oic.r.switch.binary"), which supports actuator interface ("oic.if.a") and has two possible states ("value": true/false). The switch is set to be discoverable and observable ("bm": 3; mask value 3 equals `OC_DISCOVERABLE | OC_OBSERVABLE`).

3. Use the get command again to examine the newly created switch
```bash
// retrieves all resources of all devices
./grpc -get
```

Output:
```json
...
{
    "content": {
        "content_type": "application/vnd.ocf+cbor",
        "data": "v2V2YWx1ZfT/"
    },
    "resource_id": {
        "device_id": "2b9ed3ed-ddf3-4c9c-4d21-9ec1f6ba6b03",
        "href": "/4rLN4BlwJmFmbbMJblChB2kyT2zJEP"
    },
    "status": 1,
    "types": [
        "oic.r.switch.binary"
    ]
},
...
```
