import { store, history } from '@/store'
import { Emitter } from '@shared-ui/common/services/emitter'
import { showInfoToast } from '@shared-ui/components/new/Toast'
import {
  devicesStatuses,
  resourceEventTypes,
  DEVICES_STATUS_WS_KEY,
  DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY,
} from './constants'
import {
  getDeviceNotificationKey,
  getResourceRegistrationNotificationKey,
  getResourceUpdateNotificationKey,
} from './utils'
import { isNotificationActive } from './slice'
import { getDeviceApi } from './rest'
import { messages as t } from './Devices.i18n'

const { ONLINE, REGISTERED, UNREGISTERED } = devicesStatuses
const DEFAULT_NOTIFICATION_DELAY = 500

const getDeviceIds = (deviceId, deviceRegistered, deviceUnregistered) => {
  if (deviceId) {
    return [deviceId]
  } else {
    return deviceRegistered
      ? deviceRegistered.deviceIds
      : deviceUnregistered.deviceIds
  }
}

const getEventType = deviceUnregistered =>
  deviceUnregistered ? UNREGISTERED : REGISTERED

const showToast = async (
  notificationsEnabled,
  currentDeviceNotificationsEnabled,
  deviceId,
  status
) => {
  if (
    (notificationsEnabled || currentDeviceNotificationsEnabled) &&
    status !== UNREGISTERED
  ) {
    const { data: { name } = {} } = await getDeviceApi(deviceId)
    const toastMessage =
      status === ONLINE ? t.deviceWentOnline : t.deviceWentOffline
    showInfoToast(
      {
        title: t.devicestatusChange,
        message: { message: toastMessage, params: { name } },
      },
      {
        onClick: () => {
          history.push(`/devices/${deviceId}`)
        },
        isNotification: true,
      }
    )
  }
}

// WebSocket listener for device status change.
export const deviceStatusListener = async ({
  deviceMetadataUpdated,
  deviceRegistered,
  deviceUnregistered,
}) => {
  if (deviceMetadataUpdated || deviceRegistered || deviceUnregistered) {
    const notificationsEnabled = isNotificationActive(DEVICES_STATUS_WS_KEY)(
      store.getState()
    )

    setTimeout(
      async () => {
        const {
          deviceId,
          connection: { status: deviceStatus } = {},
          twinEnabled,
        } = deviceMetadataUpdated || {}
        const eventType = getEventType(deviceUnregistered)
        const deviceIds = getDeviceIds(
          deviceId,
          deviceRegistered,
          deviceUnregistered
        )
        const status = deviceStatus || eventType

        try {
          deviceIds.forEach(async deviceId => {
            // Emit an event: things.status.{deviceId}
            Emitter.emit(`${DEVICES_STATUS_WS_KEY}.${deviceId}`, {
              deviceId,
              status,
              twinEnabled,
            })

            // Get the notification state of a single device from redux store
            const currentDeviceNotificationsEnabled = isNotificationActive(
              getDeviceNotificationKey(deviceId)
            )(store.getState())

            // Show toast
            showToast(
              notificationsEnabled,
              currentDeviceNotificationsEnabled,
              deviceId,
              status
            )
          })
        } catch (error) {} // ignore error

        // If the event was registered or unregistered, emit an event with the number to increment by
        if ([REGISTERED, UNREGISTERED].includes(status)) {
          // Emit an event: things-registered-unregistered-count
          Emitter.emit(
            DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY,
            deviceIds.length
          )
        }
      },
      notificationsEnabled ? DEFAULT_NOTIFICATION_DELAY : 0
    )
  }
}

const showToastByResources = options => {
  showInfoToast(
    {
      title: options.toastTitle,
      message: {
        message: options.toastMessage,
        params: {
          deviceName: options.deviceName,
          deviceId: options.deviceId,
          count: options.count,
          href: options.href,
        },
      },
    },
    {
      onClick: options.onClick,
      isNotification: true,
    }
  )
}

const getToastTitleByResource = (multiMode, isNew) => {
  if (multiMode) {
    return isNew ? t.newResources : t.resourcesDeleted
  } else {
    return isNew ? t.newResource : t.resourceDeleted
  }
}
const getToastMessageByResource = (multiMode, isNew) => {
  if (multiMode) {
    return isNew ? t.resourcesAdded : t.resourcesWereDeleted
  } else {
    return isNew ? t.resourceAdded : t.resourceWithHrefWasDeleted
  }
}

const getResources = (resourcePublished, resourceUnpublished) =>
  resourcePublished
    ? resourcePublished.resources // if resource was published, use the resources list from the event
    : resourceUnpublished.hrefs.map(href => ({ href })) // if the resource was unpublished, create an array of objects containing hrefs, so that it matches the resources object

export const deviceResourceRegistrationListener =
  ({ deviceId, deviceName }) =>
  ({ resourcePublished, resourceUnpublished }) => {
    if (resourcePublished || resourceUnpublished) {
      // Device notifications must be enabled to see a toast message
      const notificationsEnabled = isNotificationActive(
        getDeviceNotificationKey(deviceId)
      )(store.getState())

      const resources = getResources(resourcePublished, resourceUnpublished)
      const resourceRegistrationObservationWSKey =
        getResourceRegistrationNotificationKey(deviceId)
      const event = resourcePublished
        ? resourceEventTypes.ADDED
        : resourceEventTypes.REMOVED

      // Emit an event: things.resource.registration.{deviceId}
      Emitter.emit(`${resourceRegistrationObservationWSKey}.${event}`, {
        event,
        resources,
      })

      if (notificationsEnabled) {
        const isNew = event === resourceEventTypes.ADDED
        const toastTitle = getToastTitleByResource(resources.length >= 5, isNew)
        const toastMessage = getToastMessageByResource(
          resources.length >= 5,
          isNew
        )

        // If 5 or more resources came in the WS, show only one notification message
        if (resources.length >= 5) {
          // Show toast
          showToastByResources({
            toastTitle,
            toastMessage,
            deviceName,
            deviceId,
            count: resources.length,
            onClick: () => {
              history.push(`/devices/${deviceId}`)
            },
          })
        } else {
          resources.forEach(({ href }) => {
            showToastByResources({
              toastTitle,
              toastMessage,
              deviceName,
              deviceId,
              count: undefined,
              href: href,
              onClick: () => {
                // href -> redirect to resource and open resource modal
                // redirect to device
                history.push(`/devices/${deviceId}${isNew ? href : undefined}`)
              },
            })
          })
        }
      }
    }
  }

export const deviceResourceUpdateListener =
  ({ deviceId, href, deviceName }) =>
  ({ resourceChanged }) => {
    if (resourceChanged) {
      const eventKey = getResourceUpdateNotificationKey(deviceId, href)
      const notificationsEnabled = isNotificationActive(eventKey)(
        store.getState()
      )

      // Emit an event: things.resource.update.{deviceId}.{href}
      Emitter.emit(`${eventKey}`, resourceChanged.content)

      if (notificationsEnabled) {
        // Show toast
        showInfoToast(
          {
            title: t.resourceUpdated,
            message: {
              message: t.resourceUpdatedDesc,
              params: { href, deviceName },
            },
          },
          {
            onClick: () => {
              history.push(`/devices/${deviceId}${href}`)
            },
            isNotification: true,
          }
        )
      }
    }
  }
