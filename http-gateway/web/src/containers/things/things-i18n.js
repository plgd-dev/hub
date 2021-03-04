import { defineMessages } from '@formatjs/intl'

export const messages = defineMessages({
  online: {
    id: 'things.online',
    defaultMessage: 'online',
  },
  offline: {
    id: 'things.offline',
    defaultMessage: 'offline',
  },
  name: {
    id: 'things.name',
    defaultMessage: 'Name',
  },
  types: {
    id: 'things.types',
    defaultMessage: 'Types',
  },
  supportedTypes: {
    id: 'things.supportedTypes',
    defaultMessage: 'Supported Types',
  },
  interfaces: {
    id: 'things.interfaces',
    defaultMessage: 'Interfaces',
  },
  deviceInterfaces: {
    id: 'things.deviceInterfaces',
    defaultMessage: 'Device Interfaces',
  },
  status: {
    id: 'things.status',
    defaultMessage: 'Status',
  },
  thingNotFound: {
    id: 'things.thingNotFound',
    defaultMessage: 'Thing not found',
  },
  thingNotFoundMessage: {
    id: 'things.thingNotFoundMessage',
    defaultMessage: 'Thing with ID "{id}" does not exist.',
  },
  location: {
    id: 'things.location',
    defaultMessage: 'Location',
  },
  resources: {
    id: 'things.resources',
    defaultMessage: 'Resources',
  },
  deviceId: {
    id: 'things.deviceId',
    defaultMessage: 'Device ID',
  },
  update: {
    id: 'things.update',
    defaultMessage: 'Update',
  },
  updating: {
    id: 'things.updating',
    defaultMessage: 'Updating',
  },
  create: {
    id: 'things.create',
    defaultMessage: 'Create',
  },
  creating: {
    id: 'things.creating',
    defaultMessage: 'Creating',
  },
  retrieve: {
    id: 'things.retrieve',
    defaultMessage: 'Retrieve',
  },
  retrieving: {
    id: 'things.retrieving',
    defaultMessage: 'Retrieving',
  },
  delete: {
    id: 'things.delete',
    defaultMessage: 'Delete',
  },
  deleting: {
    id: 'things.deleting',
    defaultMessage: 'Deleting',
  },
  actions: {
    id: 'things.actions',
    defaultMessage: 'Actions',
  },
  deleteResourceMessage: {
    id: 'things.deleteResourceMessage',
    defaultMessage:
      'Are you sure you want to delete this Resource? This action cannot be undone.',
  },
  resourceWasUpdated: {
    id: 'things.resourceWasUpdated',
    defaultMessage: 'The resource was updated successfully.',
  },
  resourceWasUpdatedOffline: {
    id: 'things.resourceWasUpdatedOffline',
    defaultMessage:
      'The resource was updated successfully, changes will be applied once the device is online.',
  },
  resourceWasDeletedOffline: {
    id: 'things.resourceWasUpdatedOffline',
    defaultMessage:
      'Deleting of the resource was scheduled, but it will be executed once the device is online.',
  },
  resourceWasCreated: {
    id: 'things.resourceWasCreated',
    defaultMessage: 'The resource was created successfully.',
  },
  resourceWasCreatedOffline: {
    id: 'things.resourceWasCreatedOffline',
    defaultMessage:
      'The resource was created successfully, changes will be applied once the device is online.',
  },
  invalidArgument: {
    id: 'things.invalidArgument',
    defaultMessage: 'There was an invalid argument in the JSON structure.',
  },
  resourceUpdateSuccess: {
    id: 'things.resourceUpdateSuccess',
    defaultMessage: 'Resource update successful',
    description: 'Title of the toast message on resource update success.',
  },
  resourceUpdateError: {
    id: 'things.resourceUpdateError',
    defaultMessage: 'Failed to update a resource',
    description: 'Title of the toast message on resource update error.',
  },
  resourceCreateSuccess: {
    id: 'things.resourceCreateSuccess',
    defaultMessage: 'Resource created successfully',
    description: 'Title of the toast message on create resource success.',
  },
  resourceCreateError: {
    id: 'things.resourceCreateError',
    defaultMessage: 'Failed to create a resource',
    description: 'Title of the toast message on resource create error.',
  },
  resourceRetrieveError: {
    id: 'things.resourceRetrieveError',
    defaultMessage: 'Failed to retrieve a resource',
    description: 'Title of the toast message on resource retrieve error.',
  },
  resourceDeleteSuccess: {
    id: 'things.resourceDeleteSuccess',
    defaultMessage: 'Resource delete scheduled',
    description:
      'Title of the toast message on delete resource schedule success.',
  },
  resourceWasDeleted: {
    id: 'things.resourceWasDeleted',
    defaultMessage:
      'Resource delete was successfully scheduled. Keep the notifications turned on to see if the resource was deleted.',
  },
  resourceDeleteError: {
    id: 'things.resourceDeleteError',
    defaultMessage: 'Failed to delete a resource',
    description: 'Title of the toast message on resource delete error.',
  },
  thingWentOnline: {
    id: 'things.thingWentOnline',
    defaultMessage: 'Thing "{name}" went online.',
  },
  thingWentOffline: {
    id: 'things.thingWentOffline',
    defaultMessage: 'Thing "{name}" went offline.',
  },
  thingWasUnregistered: {
    id: 'things.thingWasUnregistered',
    defaultMessage: 'Thing "{name}" was unregistered.',
  },
  thingStatusChange: {
    id: 'things.thingStatusChange',
    defaultMessage: 'Thing status change',
  },
  notifications: {
    id: 'things.notifications',
    defaultMessage: 'Notifications',
  },
  refresh: {
    id: 'things.refresh',
    defaultMessage: 'Refresh',
  },
  newResource: {
    id: 'things.newResource',
    defaultMessage: 'New Resource',
  },
  resourceDeleted: {
    id: 'things.resourceDeleted',
    defaultMessage: 'Resource Deleted',
  },
  resourceWithHrefWasDeleted: {
    id: 'things.resourceWithHrefWasDeleted',
    defaultMessage: 'Resource {href} was deleted from thing {deviceId}.',
  },
  resourceAdded: {
    id: 'things.resourceAdded',
    defaultMessage: 'New resource {href} was added to the thing {deviceId}.',
  },
  resourceUpdated: {
    id: 'things.resourceUpdated',
    defaultMessage: 'Resource Updated',
  },
  resourceUpdatedDesc: {
    id: 'things.resourceUpdatedDesc',
    defaultMessage:
      'Resource {href} on a thing called {deviceName} was updated.',
  },
})
