// @ts-ignore
import * as converter from 'units-converter/dist/es/index'

import { Emitter } from '@shared-ui/common/services/emitter'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import { commandTypes, updatedCommandTypes, pendingCommandStatuses, NEW_PENDING_COMMAND_WS_KEY, UPDATE_PENDING_COMMANDS_WS_KEY } from './constants'
import { messages as t } from './PendingCommands.i18n'
import { messages as d } from '../Devices/Devices.i18n'
import { isNotificationActive } from '@/containers/Devices/slice'
import { getDeviceNotificationKey } from '@/containers/Devices/utils'
import { store } from '@/store'
import notificationId from '@/notificationId'

const time = converter.time

const { OK, ACCEPTED, CANCELED, CREATED, BAD_REQUEST } = pendingCommandStatuses

export const convertPendingCommandsList = (list: any) =>
    list?.map((command: any) => {
        const commandType = Object.keys(command)[0]
        const { status, ...rest } = command[commandType]

        return {
            status: status || null,
            commandType,
            ...rest,
        }
    }) || []

export const getCommandTypeFromEvent = (event: any) => Object.values(commandTypes).find((type) => event.hasOwnProperty(type))

export const getUpdatedCommandTypeFromEvent = (event: any) => Object.values(updatedCommandTypes).find((type) => event.hasOwnProperty(type))

export const createResourceNotificationId = (deviceId: string, href: string, correlationId?: string, userId?: string) => {
    let base = `deviceId=${deviceId}|href=${href}`
    if (correlationId) {
        base += `|correlationId=${correlationId}`
    }
    if (userId) {
        base += `|userId=${userId}`
    }

    return base
}

export const handleEmitNewPendingCommand = (eventData: any) => {
    const commandType = getCommandTypeFromEvent(eventData)
    const pendingCommand = commandType ? eventData?.[commandType] || null : null

    if (pendingCommand && commandType) {
        // Emit new pending command event
        Emitter.emit(NEW_PENDING_COMMAND_WS_KEY, {
            [commandType]: pendingCommand,
        })
    }
}

const DEFAULT_NOTIFICATION_DELAY = 500

export const handleEmitUpdatedCommandEvents = (eventData: any) => {
    const commandType = getUpdatedCommandTypeFromEvent(eventData)
    const updatedCommand = commandType ? eventData?.[commandType] || null : null

    if (updatedCommand) {
        const { auditContext, resourceId, deviceId, status, canceled } = updatedCommand
        const cancelBool = canceled ? CANCELED : OK

        // Emit update pending command event
        Emitter.emit(UPDATE_PENDING_COMMANDS_WS_KEY, {
            correlationId: auditContext?.correlationId,
            deviceId: resourceId?.deviceId || deviceId,
            status: typeof canceled === 'boolean' ? cancelBool : status,
        })

        const toastId = createResourceNotificationId(resourceId?.deviceId, resourceId?.href, auditContext?.correlationId, auditContext?.userId)
        const variant = isNotificationActive(getDeviceNotificationKey(deviceId))(store.getState()) ? 'toast' : 'notification'

        setTimeout(() => {
            if (!Notification.isActive(toastId) && status) {
                if (status === OK) {
                    Notification.success(
                        { title: d.resourceUpdateSuccess, message: d.resourceWasUpdated },
                        { variant, toastId, notificationId: notificationId.HUB_HANDLE_EMIT_UPDATED_COMMAND_EVENTS_OK }
                    )
                } else {
                    if (status === BAD_REQUEST) {
                        Notification.error(
                            { title: d.resourceUpdateError, message: d.invalidArgument },
                            { variant, notificationId: notificationId.HUB_HANDLE_EMIT_UPDATED_COMMAND_EVENTS_BAD_REQUEST }
                        )
                    } else {
                        Notification.error(
                            {
                                title: d.error,
                                message: d.resourceUpdateError,
                            },
                            { variant, notificationId: notificationId.HUB_HANDLE_EMIT_UPDATED_COMMAND_EVENTS }
                        )
                    }
                }
            }
        }, DEFAULT_NOTIFICATION_DELAY)
    }
}

// Updates the pending commands data with an object of
// { deviceId, href, correlationId, status } which came from the WS events.
export const updatePendingCommandsDataStatus = (
    data: any,
    { deviceId, correlationId, status }: { deviceId: string; correlationId: string; status: string }
) => {
    return data?.map((command: any) => {
        const commandType = Object.keys(command)[0]

        if (
            (command[commandType]?.resourceId?.deviceId === deviceId || command[commandType]?.deviceId === deviceId) &&
            command[commandType].auditContext.correlationId === correlationId
        ) {
            return {
                [commandType]: {
                    ...command[commandType],
                    status,
                },
            }
        }

        return command
    })
}

// validUntil - ns, currentTime - ms
export const hasCommandExpired = (validUntil: any, currentTime: any) => {
    if (validUntil === '0') return false

    const validUntilMs = time(validUntil).from('ns').to('ms').value

    return validUntilMs < currentTime
}

export const getPendingCommandStatusColorAndLabel = (status: string, validUntil: any, currentTime: any) => {
    if (!status && hasCommandExpired(validUntil, currentTime)) {
        return {
            color: 'red',
            label: t.expired,
        }
    }

    switch (status) {
        case null:
            return {
                color: 'orange',
                label: t.pending,
            }
        case OK:
        case ACCEPTED:
        case CREATED:
            return {
                color: 'green',
                label: t.successful,
            }
        case CANCELED:
            return {
                color: 'red',
                label: t.canceled,
            }
        default:
            return {
                color: 'red',
                label: t.error,
            }
    }
}
