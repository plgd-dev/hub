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
  retrieve: {
    id: 'things.retrieve',
    defaultMessage: 'Retrieve',
  },
  retrieving: {
    id: 'things.retrieving',
    defaultMessage: 'Retrieving',
  },
  resourceWasUpdated: {
    id: 'things.resourceWasUpdated',
    defaultMessage: 'Resource was updated successfully.',
  },
  resourceWasUpdatedOffline: {
    id: 'things.resourceWasUpdatedOffline',
    defaultMessage: 'Resource was updated successfully, but the changes will be applied once the device goes online.',
  },
  invalidArgument: {
    id: 'things.invalidArgument',
    defaultMessage: 'Invalid argument in the JSON structure.',
  }
})
