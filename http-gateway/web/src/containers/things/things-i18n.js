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
  thingResourcesNotFound: {
    id: 'things.thingResourcesNotFound',
    defaultMessage: 'Thing resources not found',
  },
  thingResourcesNotFoundMessage: {
    id: 'things.thingResourcesNotFoundMessage',
    defaultMessage: 'Thing resources for device with ID "{id}" does not exist.',
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
      'Are you sure you want to delete this resource? This action cannot be undone.',
  },
  resourceWasUpdated: {
    id: 'things.resourceWasUpdated',
    defaultMessage: 'The resource was updated successfully.',
  },
  resourceWasUpdatedOffline: {
    id: 'things.resourceWasUpdatedOffline',
    defaultMessage:
      'The resource update was scheduled, changes will be applied once the device is online.',
  },
  resourceWasDeletedOffline: {
    id: 'things.resourceWasDeletedOffline',
    defaultMessage:
      'Deleting of the resource was scheduled, it will be deleted once the device is online.',
  },
  resourceWasCreated: {
    id: 'things.resourceWasCreated',
    defaultMessage: 'The resource was created successfully.',
  },
  resourceWasCreatedOffline: {
    id: 'things.resourceWasCreatedOffline',
    defaultMessage:
      'The resource creation was scheduled, changes will be applied once the device is online.',
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
      'The resource delete was scheduled, you will be notified when the resource was deleted.',
  },
  resourceDeleteError: {
    id: 'things.resourceDeleteError',
    defaultMessage: 'Failed to delete a resource',
    description: 'Title of the toast message on resource delete error.',
  },
  shadowSynchronizationError: {
    id: 'things.shadowSynchronizationError',
    defaultMessage: 'Failed to set shadow synchronization',
    description:
      'Title of the toast message on shadow synchronization set error.',
  },
  shadowSynchronizationWasSetOffline: {
    id: 'things.shadowSynchronizationWasSetOffline',
    defaultMessage:
      'Shadow synchronization was scheduled, changes will be applied once the device is online.',
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
  newResources: {
    id: 'things.newResources',
    defaultMessage: 'New Resources',
  },
  resourcesDeleted: {
    id: 'things.resourcesDeleted',
    defaultMessage: 'Resources Deleted',
  },
  resourceWithHrefWasDeleted: {
    id: 'things.resourceWithHrefWasDeleted',
    defaultMessage:
      'Resource {href} was deleted from thing {deviceName} ({deviceId}).',
  },
  resourceAdded: {
    id: 'things.resourceAdded',
    defaultMessage:
      'New resource {href} was added to the thing {deviceName} ({deviceId}).',
  },
  resourcesAdded: {
    id: 'things.resourcesAdded',
    defaultMessage:
      '{count} new resources were added to the thing {deviceName} ({deviceId}).',
  },
  resourcesWereDeleted: {
    id: 'things.resourcesWereDeleted',
    defaultMessage:
      '{count} resources were deleted from thing {deviceName} ({deviceId}).',
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
  treeView: {
    id: 'things.treeView',
    defaultMessage: 'Tree view',
  },
  shadowSynchronization: {
    id: 'things.shadowSynchronization',
    defaultMessage: 'Shadow synchronization',
  },
  save: {
    id: 'things.save',
    defaultMessage: 'Save',
  },
  saving: {
    id: 'things.saving',
    defaultMessage: 'Saving',
  },
  enterThingName: {
    id: 'things.enterThingName',
    defaultMessage: 'Enter thing name',
  },
  thingNameChangeFailed: {
    id: 'things.thingNameChangeFailed',
    defaultMessage: 'Thing name change failed',
  },
  enabled: {
    id: 'things.enabled',
    defaultMessage: 'Enabled',
  },
  disabled: {
    id: 'things.disabled',
    defaultMessage: 'Disabled',
  },
  commandTimeout: {
    id: 'things.commandTimeout',
    defaultMessage: 'Command Timeout',
  },
  minimalValueIs: {
    id: 'things.minimalValueIs',
    defaultMessage: 'Minimal value is {minimalValue}.',
  },
})
