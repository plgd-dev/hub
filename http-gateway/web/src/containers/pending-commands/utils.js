import { Emitter } from '@/common/services/emitter'

import {
  commandTypes,
  updatedCommandTypes,
  pendingCommandStatuses,
  NEW_PENDING_COMMAND_WS_KEY,
  UPDATE_PENDING_COMMANDS_WS_KEY,
} from './constants'
import { messages as t } from './pending-commands-i18n'

const { UNKNOWN, OK, ACCEPTED, CANCELED, CREATED } = pendingCommandStatuses

export const convertPendingCommandsList = list =>
  list?.map(command => {
    const commandType = Object.keys(command)[0]
    const { status, ...rest } = command[commandType]

    return {
      status: status || UNKNOWN,
      commandType,
      ...rest,
    }
  }) || []

export const getCommandTypeFromEvent = event =>
  Object.values(commandTypes).find(type => event.hasOwnProperty(type))

export const getUpdatedCommandTypeFromEvent = event =>
  Object.values(updatedCommandTypes).find(type => event.hasOwnProperty(type))

export const handleEmitNewPendingCommand = eventData => {
  const commandType = getCommandTypeFromEvent(eventData)
  const pendingCommand = eventData?.[commandType] || null

  if (pendingCommand) {
    // Emit new pending command event
    Emitter.emit(NEW_PENDING_COMMAND_WS_KEY, {
      [commandType]: pendingCommand,
    })
  }
}

export const handleEmitUpdatedCommandEvents = eventData => {
  const commandType = getUpdatedCommandTypeFromEvent(eventData)
  const updatedCommand = eventData?.[commandType] || null

  if (updatedCommand) {
    const { auditContext, resourceId, status } = updatedCommand
    // Emit update pending command event
    Emitter.emit(UPDATE_PENDING_COMMANDS_WS_KEY, {
      correlationId: auditContext?.correlationId,
      deviceId: resourceId?.deviceId,
      href: resourceId?.href,
      status: status,
      commandType,
    })
  }
}

// Updates the pending commands data with an object of
// { deviceId, href, correlationId, status, commandType } which came from the WS events.
export const updatePendingCommandsDataStatus = (
  data,
  { deviceId, href, correlationId, status, commandType }
) => {
  return data?.map(command => {
    const rowCommandType = getCommandTypeFromEvent(command)

    if (
      rowCommandType === commandType &&
      command[commandType].resourceId.href === href &&
      command[commandType].resourceId.deviceId === deviceId &&
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

export const getPendingCommandStatusColorAndLabel = (
  status,
  validUntil,
  currentTime
) => {
  if (![OK, ACCEPTED, CREATED].includes(status) && validUntil < currentTime) {
    return {
      color: 'red',
      label: t.expired,
    }
  }

  switch (status) {
    case UNKNOWN:
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
