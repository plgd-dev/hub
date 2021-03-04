import { getApiErrorMessage } from '@/common/utils'
import { showErrorToast, showWarningToast } from '@/components/toast'
import {
  knownInterfaces,
  errorCodes,
  THINGS_WS_KEY,
  THINGS_RESOURCE_REGISTRATION_WS_KEY,
  THINGS_RESOURCE_UPDATE_WS_KEY,
} from './constants'
import { messages as t } from './things-i18n'

// Returns the extension for resources API for the selected interface
export const interfaceGetParam = currentInterface =>
  currentInterface ? `?interface=${currentInterface}` : ''

// Return true if a resource contains the oic.if.create interface, meaning a new resource can be created from this resource
export const canCreateResource = interfaces =>
  interfaces.includes(knownInterfaces.OIC_IF_CREATE)

// Handle the errors occured during resource update
export const handleUpdateResourceErrors = (error, isOnline, _) => {
  const errorMessage = getApiErrorMessage(error)

  if (!isOnline && errorMessage?.includes?.(errorCodes.DEADLINE_EXCEEDED)) {
    // Resource update went through, but it will be applied once the device comes online
    showWarningToast({
      title: _(t.resourceUpdateSuccess),
      message: _(t.resourceWasUpdatedOffline),
    })
  } else if (errorMessage?.includes?.(errorCodes.INVALID_ARGUMENT)) {
    // JSON validation error
    showErrorToast({
      title: _(t.resourceUpdateError),
      message: _(t.invalidArgument),
    })
  } else {
    showErrorToast({
      title: _(t.resourceUpdateError),
      message: errorMessage,
    })
  }
}

// Handle the errors occured during resource create
export const handleCreateResourceErrors = (error, isOnline, _) => {
  const errorMessage = getApiErrorMessage(error)

  if (!isOnline && errorMessage?.includes?.(errorCodes.DEADLINE_EXCEEDED)) {
    // Resource create went through, but it will be applied once the device comes online
    showWarningToast({
      title: _(t.resourceCreateSuccess),
      message: _(t.resourceWasCreatedOffline),
    })
  } else if (errorMessage?.includes?.(errorCodes.INVALID_ARGUMENT)) {
    // JSON validation error
    showErrorToast({
      title: _(t.resourceCreateError),
      message: _(t.invalidArgument),
    })
  } else {
    showErrorToast({
      title: _(t.resourceCreateError),
      message: errorMessage,
    })
  }
}

// Handle the errors occured during resource fetch
export const handleFetchResourceErrors = (error, _) =>
  showErrorToast({
    title: _(t.resourceRetrieveError),
    message: getApiErrorMessage(error),
  })

// Handle the errors occured during resource fetch
export const handleDeleteResourceErrors = (error, isOnline, _) => {
  const errorMessage = getApiErrorMessage(error)

  if (!isOnline && errorMessage?.includes?.(errorCodes.DEADLINE_EXCEEDED)) {
    // Resource update went through, but it will be applied once the device comes online
    showWarningToast({
      title: _(t.resourceDeleteSuccess),
      message: _(t.resourceWasDeletedOffline),
    })
  } else {
    showErrorToast({
      title: _(t.resourceDeleteError),
      message: errorMessage,
    })
  }
}

// Updates the device data with an object of { deviceId, status } which came from the WS events.
export const updateThingsDataStatus = (data, { deviceId, status }) => {
  return data?.map(d => {
    if (d.device.di === deviceId) {
      return {
        ...d,
        status,
      }
    }

    return d
  })
}

// Redux and event key for the notification state of a single device
export const getThingNotificationKey = deviceId =>
  `${THINGS_WS_KEY}.${deviceId}`

// Redux and event key for the notification state for a registration or unregistration of a resource
export const getResourceRegistrationNotificationKey = deviceId =>
  `${THINGS_RESOURCE_REGISTRATION_WS_KEY}.${deviceId}`

// Redux and event key for the notification state for an update of a single resource
export const getResourceUpdateNotificationKey = (deviceId, href) =>
  `${THINGS_RESOURCE_UPDATE_WS_KEY}.${deviceId}.${href}`
