import { getApiErrorMessage } from '@/common/utils'
import { showErrorToast, showWarningToast } from '@/components/toast'
import { knownInterfaces, errorCodes } from './constants'
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
