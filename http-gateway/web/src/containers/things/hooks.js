import { useState } from 'react'
import { useApi } from '@/common/hooks'
import { useAppConfig } from '@/containers/app'
import { useEmitter } from '@/common/hooks'

import { thingsApiEndpoints, THINGS_STATUS_WS_KEY } from './constants'
import { updateThingsDataWithStatuses, updateThingsStatusList } from './utils'

export const useThingsList = () => {
  const { httpGatewayAddress } = useAppConfig()
  const [updatedStatuses, setUpdatedStatuses] = useState([])

  // Fetch the data
  const { data, ...rest } = useApi(
    `${httpGatewayAddress}${thingsApiEndpoints.THINGS}`
  )

  // Update the status list when a WS event is emitted
  useEmitter(THINGS_STATUS_WS_KEY, newDeviceStatus => {
    setUpdatedStatuses(updateThingsStatusList(updatedStatuses, newDeviceStatus))
  })

  // Update the data with the current status list
  const updatedData = updateThingsDataWithStatuses(data, updatedStatuses)

  return { data: updatedData, ...rest }
}

export const useThingDetails = deviceId => {
  const { httpGatewayAddress } = useAppConfig()
  const [updatedStatus, setUpdatedStatus] = useState(null)

  // Fetch the data
  const { data, ...rest } = useApi(
    `${httpGatewayAddress}${thingsApiEndpoints.THINGS}/${deviceId}`
  )

  // Update the status when a WS event is emitted
  useEmitter(`${THINGS_STATUS_WS_KEY}.${deviceId}`, ({ status }) => {
    setUpdatedStatus(status)
  })

  // Combine the status from WS if set with the device data
  const updatedData = data
    ? {
        ...data,
        status: updatedStatus ?? data?.status,
      }
    : null

  return { data: updatedData, ...rest }
}
