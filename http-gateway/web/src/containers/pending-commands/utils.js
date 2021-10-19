import { time } from 'units-converter'

import { Emitter } from '@/common/services/emitter'

import {
  commandTypes,
  updatedCommandTypes,
  pendingCommandStatuses,
  NEW_PENDING_COMMAND_WS_KEY,
  UPDATE_PENDING_COMMANDS_WS_KEY,
} from './constants'
import { messages as t } from './pending-commands-i18n'

const { OK, ACCEPTED, CANCELED, CREATED } = pendingCommandStatuses

export const convertPendingCommandsList = list =>
  list?.map(command => {
    const commandType = Object.keys(command)[0]
    const { status, ...rest } = command[commandType]

    return {
      status: status || null,
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
    const { auditContext, resourceId, deviceId, status, canceled } =
      updatedCommand
    // Emit update pending command event
    Emitter.emit(UPDATE_PENDING_COMMANDS_WS_KEY, {
      correlationId: auditContext?.correlationId,
      deviceId: resourceId?.deviceId || deviceId,
      href: resourceId?.href,
      status: typeof canceled === 'boolean' ? (canceled ? CANCELED : OK) : status,
    })
  }
}

// Updates the pending commands data with an object of
// { deviceId, href, correlationId, status } which came from the WS events.
export const updatePendingCommandsDataStatus = (
  data,
  { deviceId, href, correlationId, status }
) => {
  return data?.map(command => {
    const commandType = Object.keys(command)[0]

    if (
      command[commandType]?.resourceId?.href === href &&
      (command[commandType]?.resourceId?.deviceId === deviceId ||
        command[commandType]?.deviceId === deviceId) &&
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
export const hasCommandExpired = (validUntil, currentTime) => {
  if (validUntil === '0') return false

  const validUntilMs = time(validUntil).from('ns').to('ms').value

  return validUntilMs < currentTime
}

export const getPendingCommandStatusColorAndLabel = (
  status,
  validUntil,
  currentTime
) => {
  if (
    ![OK, ACCEPTED, CREATED].includes(status) &&
    hasCommandExpired(validUntil, currentTime)
  ) {
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
