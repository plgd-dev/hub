import { store, history } from '@/store'

import { Emitter } from '@shared-ui/common/services/emitter'
import { notifications } from '@shared-ui/common/services'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import { devicesStatuses, resourceEventTypes, DEVICES_STATUS_WS_KEY, DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY } from './constants'
import { getDeviceNotificationKey, getResourceRegistrationNotificationKey, getResourceUpdateNotificationKey } from './utils'
import { isNotificationActive } from './slice'
import { getDeviceApi } from './rest'
import { messages as t } from './Devices.i18n'
import notificationId from '@/notificationId'

const { ONLINE, REGISTERED, UNREGISTERED, OFFLINE } = devicesStatuses
const DEFAULT_NOTIFICATION_DELAY = 500

const getDeviceIds = (deviceId, deviceRegistered, deviceUnregistered) => {
    if (deviceId) {
        return [deviceId]
    } else {
        return deviceRegistered ? deviceRegistered.deviceIds : deviceUnregistered.deviceIds
    }
}

const getEventType = (deviceUnregistered) => (deviceUnregistered ? UNREGISTERED : REGISTERED)

const showToast = async (currentDeviceNotificationsEnabled, deviceId, status) => {
    if (status !== UNREGISTERED) {
        const { data: { name } = {} } = await getDeviceApi(deviceId)

        const getToastMessage = () => {
            switch (status) {
                case OFFLINE:
                    return { message: t.deviceWentOffline, params: { name } }
                case REGISTERED:
                    return { message: t.deviceWasRegistered, params: { name } }
                case ONLINE:
                default:
                    return { message: t.deviceWentOnline, params: { name } }
            }
        }

        Notification.info(
            {
                title: t.devicestatusChange,
                message: [ONLINE, ONLINE, REGISTERED].includes(status) ? getToastMessage() : `Device state: ${status}`,
            },
            {
                variant: currentDeviceNotificationsEnabled ? 'toast' : 'notification',
                onClick: () => {
                    history.push(`/devices/${deviceId}`)
                },
                toastId: `${deviceId}|status|${status}`,
                notificationId: notificationId.HUB_SHOW_TOAST,
            }
        )
    }
}

// WebSocket listener for device status change.
export const deviceStatusListener = async (props) => {
    const { deviceMetadataUpdated, deviceRegistered, deviceUnregistered } = props
    if (deviceMetadataUpdated || deviceRegistered || deviceUnregistered) {
        // const notificationsEnabled = isNotificationActive(DEVICES_STATUS_WS_KEY)(store.getState())

        const { deviceId, connection: { status: deviceStatus } = {}, twinEnabled } = deviceMetadataUpdated || {}
        const eventType = getEventType(deviceUnregistered)
        const status = deviceStatus || eventType
        const deviceIds = getDeviceIds(deviceId, deviceRegistered, deviceUnregistered)

        if ([REGISTERED, UNREGISTERED].includes(status)) {
            // If the event was registered or unregistered, emit an event with the number to increment by
            Emitter.emit(DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY, deviceIds.length)
        }

        setTimeout(() => {
            try {
                deviceIds.forEach(async (deviceId) => {
                    // Emit an event: devices.status.{deviceId}
                    Emitter.emit(`${DEVICES_STATUS_WS_KEY}.${deviceId}`, {
                        deviceId,
                        status,
                        twinEnabled,
                    })

                    if (status !== UNREGISTERED) {
                        const lastNotification = notifications.getLastNotification(deviceId)

                        // show toast only if last change ( status ) is different from prev
                        if (!lastNotification || lastNotification.type !== `status-${status}`) {
                            notifications.addNotification({
                                deviceId,
                                type: `status-${status}`,
                            })

                            // Get the notification state of a single device from redux store
                            const currentDeviceNotificationsEnabled = isNotificationActive(getDeviceNotificationKey(deviceId))(store.getState())

                            await showToast(currentDeviceNotificationsEnabled, deviceId, status).then()
                        }
                    }
                })
            } catch (error) {} // ignore error
        }, DEFAULT_NOTIFICATION_DELAY)
    }
}

const showToastByResources = (options) => {
    Notification.info(
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
            notificationId: notificationId.HUB_SHOW_TOAST_BY_RESOURCES,
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
        : resourceUnpublished.hrefs.map((href) => ({ href })) // if the resource was unpublished, create an array of objects containing hrefs, so that it matches the resources object

export const deviceResourceRegistrationListener =
    ({ deviceId, deviceName }) =>
    ({ resourcePublished, resourceUnpublished }) => {
        if (resourcePublished || resourceUnpublished) {
            // Device notifications must be enabled to see a toast message
            const notificationsEnabled = isNotificationActive(getDeviceNotificationKey(deviceId))(store.getState())

            const resources = getResources(resourcePublished, resourceUnpublished)
            const resourceRegistrationObservationWSKey = getResourceRegistrationNotificationKey(deviceId)
            const event = resourcePublished ? resourceEventTypes.ADDED : resourceEventTypes.REMOVED

            // Emit an event: things.resource.registration.{deviceId}
            Emitter.emit(`${resourceRegistrationObservationWSKey}.${event}`, {
                event,
                resources,
            })

            if (notificationsEnabled) {
                const isNew = event === resourceEventTypes.ADDED
                const toastTitle = getToastTitleByResource(resources.length >= 5, isNew)
                const toastMessage = getToastMessageByResource(resources.length >= 5, isNew)

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
            const notificationsEnabled = isNotificationActive(eventKey)(store.getState())

            // Emit an event: things.resource.update.{deviceId}.{href}
            Emitter.emit(`${eventKey}`, resourceChanged.content)

            if (notificationsEnabled) {
                // Show toast
                Notification.info(
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
                        notificationId: notificationId.HUB_DEVICE_RESOURCE_UPDATE_LISTENER,
                    }
                )
            }
        }
    }
