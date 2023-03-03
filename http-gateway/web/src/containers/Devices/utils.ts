// @ts-ignore
import * as converter from 'units-converter/dist/es/index'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import {
  showErrorToast,
  showWarningToast,
} from '@shared-ui/components/new/Toast'
import { compareIgnoreCase } from '@shared-ui/components/new/Table/Utils'
import { errorCodes } from '@shared-ui/common/services/fetch-api'
import {
  knownInterfaces,
  knownResourceTypes,
  DEVICES_WS_KEY,
  DEVICES_RESOURCE_REGISTRATION_WS_KEY,
  DEVICES_RESOURCE_UPDATE_WS_KEY,
  commandTimeoutUnits,
  MINIMAL_TTL_VALUE_MS, devicesStatuses
} from "./constants";
import { messages as t } from './Devices.i18n'
import { DeviceDataType, ResourcesType } from "@/containers/Devices/Devices.types";

const time = converter.time

const { INFINITE, NS, MS, S, M, H } = commandTimeoutUnits

// Returns the extension for resources API for the selected interface
export const interfaceGetParam = (
  currentInterface: string | null,
  join = '?'
) =>
  currentInterface && currentInterface !== ''
    ? `${join}resourceInterface=${currentInterface}`
    : ''

// Return true if a resource contains the oic.if.create interface, meaning a new resource can be created from this resource
export const canCreateResource = (interfaces: string[]) =>
  interfaces.includes(knownInterfaces.OIC_IF_CREATE)

// Returns true if a device has a resource oic.wk.con which holds the device name property
export const canChangeDeviceName = (links: ResourcesType[]) =>
  links.findIndex(link =>
    link.resourceTypes.includes(knownResourceTypes.OIC_WK_CON)
  ) !== -1

// Returns the href for the resource which can do a device name change
export const getDeviceChangeResourceHref = (links: ResourcesType[]) =>
  links.find(link => link.resourceTypes.includes(knownResourceTypes.OIC_WK_CON))
    ?.href

// Handle the errors occurred during resource update
export const handleUpdateResourceErrors = (
  error: any,
  { id: deviceId, href }: { id: string; href: string },
  _: any
) => {
  const errorMessage = getApiErrorMessage(error)

  if (errorMessage?.includes?.(errorCodes.DEADLINE_EXCEEDED)) {
    // Resource update went through, but it will be applied once the device comes online
    showWarningToast({
      title: _(t.resourceUpdate),
      message: _(t.resourceWasUpdatedOffline),
    })
  } else if (errorMessage?.includes?.(errorCodes.COMMAND_EXPIRED)) {
    // Command timeout
    showWarningToast({
      title: _(t.resourceUpdate),
      message: `${_(t.update)} ${_(t.commandOnResourceExpired, {
        deviceId,
        href,
      })}`,
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

// Handle the errors occurred during resource create
export const handleCreateResourceErrors = (
  error: any,
  { id: deviceId, href }: { id: string; href: string },
  _: any
) => {
  const errorMessage = getApiErrorMessage(error)

  if (errorMessage?.includes?.(errorCodes.DEADLINE_EXCEEDED)) {
    // Resource create went through, but it will be applied once the device comes online
    showWarningToast({
      title: _(t.resourceCreate),
      message: _(t.resourceWasCreatedOffline),
    })
  } else if (errorMessage?.includes?.(errorCodes.COMMAND_EXPIRED)) {
    // Command timeout
    showWarningToast({
      title: _(t.resourceCreate),
      message: `${_(t.create)} ${_(t.commandOnResourceExpired, {
        deviceId,
        href,
      })}`,
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

// Handle the errors occurred twinSynchronization set
export const handleTwinSynchronizationErrors = (error: any, _: any) => {
  const errorMessage = getApiErrorMessage(error)

  if (errorMessage?.includes?.(errorCodes.DEADLINE_EXCEEDED)) {
    // Twin synchronization set went through, but it will be applied once the device comes online
    showWarningToast({
      title: _(t.twinSynchronization),
      message: _(t.twinSynchronizationWasSetOffline),
    })
  } else {
    showErrorToast({
      title: _(t.twinSynchronizationError),
      message: errorMessage,
    })
  }
}

// Handle the errors occurred during resource fetch
export const handleFetchResourceErrors = (error: any, _: any) =>
  showErrorToast({
    title: _(t.resourceRetrieveError),
    message: getApiErrorMessage(error),
  })

// Handle the errors occurred during resource fetch
export const handleDeleteResourceErrors = (
  error: any,
  { id: deviceId, href }: { id: string; href: string },
  _: any
) => {
  const errorMessage = getApiErrorMessage(error)

  if (errorMessage?.includes?.(errorCodes.DEADLINE_EXCEEDED)) {
    // Resource update went through, but it will be applied once the device comes online
    showWarningToast({
      title: _(t.resourceDelete),
      message: _(t.resourceWasDeletedOffline),
    })
  } else if (errorMessage?.includes?.(errorCodes.COMMAND_EXPIRED)) {
    // Command timeout
    showWarningToast({
      title: _(t.resourceDelete),
      message: `${_(t.delete)} ${_(t.commandOnResourceExpired, {
        deviceId,
        href,
      })}`,
    })
  } else {
    showErrorToast({
      title: _(t.resourceDeleteError),
      message: errorMessage,
    })
  }
}

// Handle the errors occurred during devices delete
export const handleDeleteDevicesErrors = (
  error: any,
  _: any,
  singular = false
) => {
  const errorMessage = getApiErrorMessage(error)

  showErrorToast({
    title: !singular ? _(t.devicesDeletionError) : _(t.deviceDeletionError),
    message: errorMessage,
  })
}

// Updates the device data with an object of { deviceId, status, twinEnabled } which came from the WS events.
export const updateDevicesDataStatus = (
  data: any,
  {
    deviceId,
    status,
    twinEnabled,
  }: { deviceId: string; status: string; twinEnabled: boolean }
) => {
  return data?.map((device: any) => {
    if (device.id === deviceId) {
      return {
        ...device,
        metadata: {
          ...device.metadata,
          twinEnabled,
          connection: {
            ...device.metadata.connection,
            status: status,
          },
        },
      }
    }

    return device
  })
}

// Async function for waiting
export const sleep = (ms: number) =>
  new Promise(resolve => setTimeout(resolve, ms))

/** Tree Structure utilities **/
// Shout out to @oskarbauer for creating this script :)

// A recursive function which "densify" the subRows
const deDensisfy = (objectToDeDensify: any) => {
  const { href, ...rest } = objectToDeDensify

  const keys = Object.keys(rest)
  return keys
    .map(thisKey => {
      const value = objectToDeDensify[thisKey]
      if (value.subRows) {
        value.subRows = deDensisfy(value.subRows)
      }
      return value
    })
    .sort((a, b) => {
      return compareIgnoreCase(a.href, b.href)
    })
}

// A recursive function for creating a tree structure from the href attribute
const addItem = (objToAddTo: any, item: any, position: number) => {
  const { href, ...rest } = item
  const parts = href.split('/')
  const isLast = position === parts.length - 1
  position = position + 1
  const key = `/${parts.slice(1, position).join('/')}/`

  if (isLast) {
    objToAddTo[key] = { ...objToAddTo[key], ...rest, href: key }
  } else {
    objToAddTo[key] = {
      ...objToAddTo[key],
      ...(key === href ? rest : {}),
      href: key,
      subRows: { ...(objToAddTo[key]?.subRows || {}) }, // subRows is the next level in the tree structure
    }
    // Go deeper with recursion
    addItem(objToAddTo[key].subRows, item, position)
  }
}

export const createNestedResourceData = (data: any) => {
  // Always construct the objects from scratch
  let firstSwipe = {}
  if (data) {
    data.forEach((item: any) => {
      addItem(firstSwipe, item, 1)
    })
  }
  // Then take the object structure and output the tree scructure
  const output = deDensisfy(firstSwipe)

  // Finally sort the output by href
  return output.sort((a, b) => {
    return compareIgnoreCase(a.href, b.href)
  })
}
/** End **/

// Returns the last section of a resource href, no matter if it ends with a trailing slash or not
export const getLastPartOfAResourceHref = (href: string) => {
  if (!href) {
    return ''
  }
  const values = href.split('/').filter(_t => _t)
  return values[values.length - 1]
}

// Converts a value to ns (if the unit is Infinite, it defaults to ns)
export const convertValueToNs = (value: number, unit: string) =>
  +time(value)
    .from(unit === INFINITE ? NS : unit)
    .to(NS)
    .value.toFixed(0)

// Converts a value from a given unit to a provided unit (if the unit is Infinite, it defaults to ns)
export const convertValueFromTo = (
  value: number,
  unitFrom: string,
  unitTo: string
) =>
  time(value)
    .from(unitFrom === INFINITE ? NS : unitFrom)
    .to(unitTo === INFINITE ? NS : unitTo).value

// Normalizes a given value to a fixed float number
export const normalizeToFixedFloatValue = (value: any) => +value.toFixed(5)

// Return a unit for the value which is the "nicest" after a conversion from ns
export const findClosestUnit = (value: number) => {
  const fromValue = time(value).from(NS)

  if (fromValue.to(MS).value < 1000) {
    return MS
  } else if (fromValue.to(S).value < 60) {
    return S
  } else if (fromValue.to(M).value < 60) {
    return M
  } else {
    return H
  }
}

// Return true if there is a command timeout error based on the provided value and unit
export const hasCommandTimeoutError = (value: number, unit: string) => {
  const baseUnit = unit === INFINITE ? NS : unit

  const valueMs = time(value).from(baseUnit).to(MS).value
  return valueMs < MINIMAL_TTL_VALUE_MS && value !== 0
}

export const convertAndNormalizeValueFromTo = (
  value: number,
  unitFrom: string,
  unitTo: string
) => normalizeToFixedFloatValue(convertValueFromTo(value, unitFrom, unitTo))

// Redux and event key for the notification state of a single device
export const getDeviceNotificationKey = (deviceId: string) =>
  `${DEVICES_WS_KEY}.${deviceId}`

// Redux and event key for the notification state for a registration or unregistration of a resource
export const getResourceRegistrationNotificationKey = (deviceId: string) =>
  `${DEVICES_RESOURCE_REGISTRATION_WS_KEY}.${deviceId}`

// Redux and event key for the notification state for an update of a single resource
export const getResourceUpdateNotificationKey = (
  deviceId: string,
  href: string
) => `${DEVICES_RESOURCE_UPDATE_WS_KEY}.${deviceId}.${href}`

export const isDeviceOnline = (data: DeviceDataType) => {
  const untilDate = new Date((data?.metadata?.connection?.onlineValidUntil || 0) / 100000)
  const now = new Date()
  const deviceStatus = data?.metadata?.connection?.status

  return untilDate > now && devicesStatuses.ONLINE === deviceStatus
}