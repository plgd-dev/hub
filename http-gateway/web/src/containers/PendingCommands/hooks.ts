import { useStreamApi } from '@/common/hooks'
import { useEmitter } from '@/common/hooks'

import {
  pendingCommandsApiEndpoints,
  NEW_PENDING_COMMAND_WS_KEY,
  UPDATE_PENDING_COMMANDS_WS_KEY,
} from './constants'
import {
  convertPendingCommandsList,
  updatePendingCommandsDataStatus,
} from './utils'
import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { security } from '@/common/services'

const getConfig = () => security.getGeneralConfig() as SecurityConfig

export const usePendingCommandsList = (deviceId: string) => {
  // Fetch the data
  const { data, updateData, ...rest }: StreamApiPropsType = useStreamApi(
    `${getConfig()}${pendingCommandsApiEndpoints.PENDING_COMMANDS}${
      deviceId ? `?deviceIdFilter=${deviceId}` : ''
    }`,
    {
      telemetrySpan: 'get-pending-commands',
    }
  )

  // Add a new pending command when a WS event is emitted
  useEmitter(NEW_PENDING_COMMAND_WS_KEY, (newCommand: string) => {
    updateData((data || []).concat(newCommand))
  })

  useEmitter(
    UPDATE_PENDING_COMMANDS_WS_KEY,
    (updated: {
      deviceId: string
      href: string
      correlationId: string
      status: string
    }) => {
      updateData(updatePendingCommandsDataStatus(data, updated))
    }
  )

  return {
    data: convertPendingCommandsList(data),
    updateData,
    ...rest,
  }
}
