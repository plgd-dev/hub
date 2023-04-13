import { defineMessages } from '@formatjs/intl'

export const messages = defineMessages({
    online: {
        id: 'devices.online',
        defaultMessage: 'online',
    },
    offline: {
        id: 'devices.offline',
        defaultMessage: 'offline',
    },
    device: {
        id: 'devices.device',
        defaultMessage: 'Device',
    },
    addDevice: {
        id: 'devices.addDevice',
        defaultMessage: 'Add Device',
    },
    name: {
        id: 'devices.name',
        defaultMessage: 'Name',
    },
    supportedTypes: {
        id: 'devices.supportedTypes',
        defaultMessage: 'Supported Types',
    },
    interfaces: {
        id: 'devices.interfaces',
        defaultMessage: 'Interfaces',
    },
    resourceInterfaces: {
        id: 'devices.resourceInterfaces',
        defaultMessage: 'Resource Interfaces',
    },
    deviceNotFound: {
        id: 'devices.deviceNotFound',
        defaultMessage: 'device not found',
    },
    deviceNotFoundMessage: {
        id: 'devices.deviceNotFoundMessage',
        defaultMessage: 'device with ID "{id}" does not exist.',
    },
    deviceResourcesNotFound: {
        id: 'devices.deviceResourcesNotFound',
        defaultMessage: 'device resources not found',
    },
    deviceResourcesNotFoundMessage: {
        id: 'devices.deviceResourcesNotFoundMessage',
        defaultMessage: 'device resources for device with ID "{id}" does not exist.',
    },
    href: {
        id: 'devices.href',
        defaultMessage: 'Href',
    },
    resources: {
        id: 'devices.resources',
        defaultMessage: 'Resources',
    },
    deviceId: {
        id: 'devices.deviceId',
        defaultMessage: 'Device ID',
    },
    update: {
        id: 'devices.update',
        defaultMessage: 'Update',
    },
    updating: {
        id: 'devices.updating',
        defaultMessage: 'Updating',
    },
    create: {
        id: 'devices.create',
        defaultMessage: 'Create',
    },
    creating: {
        id: 'devices.creating',
        defaultMessage: 'Creating',
    },
    details: {
        id: 'devices.details',
        defaultMessage: 'Details',
    },
    retrieve: {
        id: 'devices.retrieve',
        defaultMessage: 'Retrieve',
    },
    retrieving: {
        id: 'devices.retrieving',
        defaultMessage: 'Retrieving',
    },
    delete: {
        id: 'devices.delete',
        defaultMessage: 'Delete',
    },
    deleting: {
        id: 'devices.deleting',
        defaultMessage: 'Deleting',
    },
    action: {
        id: 'devices.action',
        defaultMessage: 'Action',
    },
    actions: {
        id: 'devices.actions',
        defaultMessage: 'Actions',
    },
    deleteResourceMessage: {
        id: 'devices.deleteResourceMessage',
        defaultMessage: 'Are you sure you want to delete this resource?',
    },
    deleteResourceMessageSubtitle: {
        id: 'devices.deleteResourceMessageSubtitle',
        defaultMessage: 'This action cannot be undone.',
    },
    deleteDeviceMessage: {
        id: 'devices.deleteDeviceMessage',
        defaultMessage: 'Are you sure you want to delete this device?',
    },
    deleteDeviceMessageSubTitle: {
        id: 'devices.deleteDeviceMessageSubTitle',
        defaultMessage: 'This action cannot be undone.',
    },
    deleteDevicesMessage: {
        id: 'devices.deleteDevicesMessage',
        defaultMessage: 'Are you sure you want to delete {count} devices?',
    },
    resourceWasUpdated: {
        id: 'devices.resourceWasUpdated',
        defaultMessage: 'The resource was updated successfully.',
    },
    resourceWasUpdatedOffline: {
        id: 'devices.resourceWasUpdatedOffline',
        defaultMessage: 'The resource update was scheduled, changes will be applied once the device is online.',
    },
    resourceWasDeletedOffline: {
        id: 'devices.resourceWasDeletedOffline',
        defaultMessage: 'Deleting of the resource was scheduled, it will be deleted once the device is online.',
    },
    resourceWasCreated: {
        id: 'devices.resourceWasCreated',
        defaultMessage: 'The resource was created successfully.',
    },
    resourceWasCreatedOffline: {
        id: 'devices.resourceWasCreatedOffline',
        defaultMessage: 'The resource creation was scheduled, changes will be applied once the device is online.',
    },
    invalidArgument: {
        id: 'devices.invalidArgument',
        defaultMessage: 'There was an invalid argument in the JSON structure.',
    },
    resourceUpdateSuccess: {
        id: 'devices.resourceUpdateSuccess',
        defaultMessage: 'Resource update successful',
        description: 'Title of the toast message on resource update success.',
    },
    resourceUpdate: {
        id: 'devices.resourceUpdate',
        defaultMessage: 'Resource update',
        description: 'Title of the toast message on resource update expired.',
    },
    resourceCreate: {
        id: 'devices.resourceCreate',
        defaultMessage: 'Resource creation',
        description: 'Title of the toast message on resource creation expired.',
    },
    resourceDelete: {
        id: 'devices.resourceDelete',
        defaultMessage: 'Resource deletion',
        description: 'Title of the toast message on resource deletion expired.',
    },
    commandOnResourceExpired: {
        id: 'devices.commandOnResourceExpired',
        defaultMessage: 'command on resource {deviceId}{href} has expired.',
        description: 'Continuos message for command expiration, keep the first letter lowercase!',
    },
    resourceUpdateError: {
        id: 'devices.resourceUpdateError',
        defaultMessage: 'Failed to update a resource',
        description: 'Title of the toast message on resource update error.',
    },
    resourceCreateSuccess: {
        id: 'devices.resourceCreateSuccess',
        defaultMessage: 'Resource created successfully',
        description: 'Title of the toast message on create resource success.',
    },
    resourceCreateError: {
        id: 'devices.resourceCreateError',
        defaultMessage: 'Failed to create a resource',
        description: 'Title of the toast message on resource create error.',
    },
    resourceRetrieveError: {
        id: 'devices.resourceRetrieveError',
        defaultMessage: 'Failed to retrieve a resource',
        description: 'Title of the toast message on resource retrieve error.',
    },
    resourceDeleteSuccess: {
        id: 'devices.resourceDeleteSuccess',
        defaultMessage: 'Resource delete scheduled',
        description: 'Title of the toast message on delete resource schedule success.',
    },
    resourceWasDeleted: {
        id: 'devices.resourceWasDeleted',
        defaultMessage: 'The resource delete was scheduled, you will be notified when the resource was deleted.',
    },
    resourceDeleteError: {
        id: 'devices.resourceDeleteError',
        defaultMessage: 'Failed to delete a resource',
        description: 'Title of the toast message on resource delete error.',
    },
    twinSynchronizationError: {
        id: 'devices.twinSynchronizationError',
        defaultMessage: 'Failed to set twin synchronization',
        description: 'Title of the toast message on twin synchronization set error.',
    },
    twinSynchronizationWasSetOffline: {
        id: 'devices.twinSynchronizationWasSetOffline',
        defaultMessage: 'Twin synchronization was scheduled, changes will be applied once the device is online.',
    },
    deviceWentOnline: {
        id: 'devices.deviceWentOnline',
        defaultMessage: 'Device "{name}" went online.',
    },
    deviceWentOffline: {
        id: 'devices.deviceWentOffline',
        defaultMessage: 'Device "{name}" went offline.',
    },
    deviceWasUnregistered: {
        id: 'devices.deviceWasUnregistered',
        defaultMessage: 'Device "{name}" was unregistered.',
    },
    devicestatusChange: {
        id: 'devices.devicestatusChange',
        defaultMessage: 'Device status change',
    },
    notifications: {
        id: 'devices.notifications',
        defaultMessage: 'Notifications',
    },
    refresh: {
        id: 'devices.refresh',
        defaultMessage: 'Refresh',
    },
    newResource: {
        id: 'devices.newResource',
        defaultMessage: 'New Resource',
    },
    resourceDeleted: {
        id: 'devices.resourceDeleted',
        defaultMessage: 'Resource Deleted',
    },
    newResources: {
        id: 'devices.newResources',
        defaultMessage: 'New Resources',
    },
    resourcesDeleted: {
        id: 'devices.resourcesDeleted',
        defaultMessage: 'Resources Deleted',
    },
    resourceWithHrefWasDeleted: {
        id: 'devices.resourceWithHrefWasDeleted',
        defaultMessage: 'Resource {href} was deleted from device {deviceName} ({deviceId}).',
    },
    resourceAdded: {
        id: 'devices.resourceAdded',
        defaultMessage: 'New resource {href} was added to the device {deviceName} ({deviceId}).',
    },
    resourcesAdded: {
        id: 'devices.resourcesAdded',
        defaultMessage: '{count} new resources were added to the device {deviceName} ({deviceId}).',
    },
    resourcesWereDeleted: {
        id: 'devices.resourcesWereDeleted',
        defaultMessage: '{count} resources were deleted from device {deviceName} ({deviceId}).',
    },
    resourceUpdated: {
        id: 'devices.resourceUpdated',
        defaultMessage: 'Resource Updated',
    },
    resourceUpdatedDesc: {
        id: 'devices.resourceUpdatedDesc',
        defaultMessage: 'Resource {href} on a device called {deviceName} was updated.',
    },
    treeView: {
        id: 'devices.treeView',
        defaultMessage: 'Tree view',
    },
    twinSynchronization: {
        id: 'devices.twinSynchronization',
        defaultMessage: 'Twin synchronization',
    },
    save: {
        id: 'devices.save',
        defaultMessage: 'Save',
    },
    saving: {
        id: 'devices.saving',
        defaultMessage: 'Saving',
    },
    deviceNameChangeFailed: {
        id: 'devices.deviceNameChangeFailed',
        defaultMessage: 'device name change failed',
    },
    enabled: {
        id: 'devices.enabled',
        defaultMessage: 'Enabled',
    },
    disabled: {
        id: 'devices.disabled',
        defaultMessage: 'Disabled',
    },
    minimalValueIs: {
        id: 'devices.minimalValueIs',
        defaultMessage: 'Minimal value is {minimalValue}.',
    },
    devicesDeleted: {
        id: 'devices.devicesDeleted',
        defaultMessage: 'devices deleted',
        description: 'Title of the toast message on devices deleted success.',
    },
    devicesDeletedMessage: {
        id: 'devices.devicesDeletedMessage',
        defaultMessage: 'The selected devices were successfully deleted.',
    },
    deviceDeleted: {
        id: 'devices.deviceDeleted',
        defaultMessage: 'device deleted',
        description: 'Title of the toast message on device deleted success.',
    },
    deviceWasDeleted: {
        id: 'devices.deviceWasDeleted',
        defaultMessage: 'device {name} was successfully deleted.',
    },
    devicesDeletionError: {
        id: 'devices.devicesDeletion',
        defaultMessage: 'Failed to delete selected devices.',
        description: 'Title of the toast message on devices deleted failed.',
    },
    deviceDeletionError: {
        id: 'devices.deviceDeletionError',
        defaultMessage: 'Failed to delete this device.',
        description: 'Title of the toast message on devices deleted failed.',
    },
    default: {
        id: 'devices.default',
        defaultMessage: 'Default',
    },
    cancel: {
        id: 'devices.cancel',
        defaultMessage: 'Cancel',
    },
    close: {
        id: 'devices.close',
        defaultMessage: 'Close',
    },
    enterDeviceId: {
        id: 'devices.enterDeviceId',
        defaultMessage: 'Enter the device ID',
    },
    getCode: {
        id: 'devices.getCode',
        defaultMessage: 'Get the Code',
    },
    back: {
        id: 'devices.back',
        defaultMessage: 'Back',
    },
    provisionNewDevice: {
        id: 'devices.provisionNewDevice',
        defaultMessage: 'Provision a new device',
    },
    deviceAuthCodeError: {
        id: 'devices.deviceAuthCodeError',
        defaultMessage: 'Device Authorization Code Error',
    },
    authorizationCode: {
        id: 'devices.authorizationCode',
        defaultMessage: 'Authorization Code',
    },
    authorizationProvider: {
        id: 'devices.authorizationProvider',
        defaultMessage: 'Authorization Provider',
    },
    deviceEndpoint: {
        id: 'devices.deviceEndpoint',
        defaultMessage: 'Device Endpoint',
    },
    hubId: {
        id: 'devices.hubId',
        defaultMessage: 'Hub ID',
    },
    certificateAuthorities: {
        id: 'devices.certificateAuthorities',
        defaultMessage: 'Certificate Authorities',
    },
    enterDeviceName: {
        id: 'devices.enterDeviceName',
        defaultMessage: 'Enter device name',
    },
    twinState: {
        id: 'devices.twinState',
        defaultMessage: 'Twin state',
    },
    subscribeNotify: {
        id: 'devices.subscribeNotify',
        defaultMessage: 'Subscribe & notify',
    },
    logging: {
        id: 'devices.logging',
        defaultMessage: 'Logging',
    },
    id: {
        id: 'devices.id',
        defaultMessage: 'ID',
    },
    model: {
        id: 'devices.model',
        defaultMessage: 'Model',
    },
    types: {
        id: 'devices.types',
        defaultMessage: 'Types',
    },
    firmware: {
        id: 'devices.firmware',
        defaultMessage: 'Firmware',
    },
    status: {
        id: 'devices.status',
        defaultMessage: 'Status',
    },
    on: {
        id: 'devices.on',
        defaultMessage: 'On',
    },
    off: {
        id: 'devices.off',
        defaultMessage: 'Off',
    },
    commandTimeout: {
        id: 'devices.commandTimeout',
        defaultMessage: 'Command Timeout',
    },
    deviceName: {
        id: 'devices.deviceName',
        defaultMessage: 'Device Name',
    },
    view: {
        id: 'devices.view',
        defaultMessage: 'View',
    },
    enterDeviceID: {
        id: 'devices.enterDeviceID',
        defaultMessage: 'Enter the device ID',
    },
    invalidUuidFormat: {
        id: 'devices.invalidUuidFormat',
        defaultMessage: 'Invalid uuid format',
    },
    getTheCode: {
        id: 'devices.getTheCode',
        defaultMessage: 'Get the code',
    },
    deviceInformation: {
        id: 'devices.deviceInformation',
        defaultMessage: 'Device information',
    },
    search: {
        id: 'devices.search',
        defaultMessage: 'Search',
    },
    twinUpdateMessage: {
        id: 'devices.twinUpdate',
        defaultMessage: 'The ongoing status change in the "Twin State" may take a while.',
    },
    edit: {
        id: 'devices.edit',
        defaultMessage: 'Edit',
    },
    editName: {
        id: 'devices.editName',
        defaultMessage: 'Edit name',
    },
    copy: {
        id: 'devices.copy',
        defaultMessage: 'Copy to clipboard',
    },
    reset: {
        id: 'devices.reset',
        defaultMessage: 'Reset',
    },
    saveChange: {
        id: 'devices.saveChange',
        defaultMessage: 'Save change',
    },
    savingChanges: {
        id: 'devices.savingChanges',
        defaultMessage: 'Saving change',
    },
    recentTasks: {
        id: 'pendingCommands.recentTasks',
        defaultMessage: 'Recent tasks',
    },
})
