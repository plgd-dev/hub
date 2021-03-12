import debounce from 'lodash/debounce'

import { useApi } from '@/common/hooks'
import { useAppConfig } from '@/containers/app'
import { useEmitter } from '@/common/hooks'

import {
  thingsApiEndpoints,
  THINGS_STATUS_WS_KEY,
  resourceEventTypes,
} from './constants'
import {
  updateThingsDataStatus,
  getResourceRegistrationNotificationKey,
} from './utils'

export const useThingsList = () => {
  const { httpGatewayAddress } = useAppConfig()

  // Fetch the data
  const { data, updateData, ...rest } = useApi(
    `${httpGatewayAddress}${thingsApiEndpoints.THINGS}`
  )

  // Update the status list when a WS event is emitted
  useEmitter(THINGS_STATUS_WS_KEY, newDeviceStatus => {
    if (data) {
      // Update the data with the current device status
      updateData(updateThingsDataStatus(data, newDeviceStatus))
    }
  })

  return { data, updateData, ...rest }
}

export const useThingDetails = deviceId => {
  const { httpGatewayAddress } = useAppConfig()

  // Fetch the data
  const { data, updateData, ...rest } = useApi(
    `${httpGatewayAddress}${thingsApiEndpoints.THINGS}/${deviceId}`
  )

  // Update the status when a WS event is emitted
  useEmitter(
    `${THINGS_STATUS_WS_KEY}.${deviceId}`,
    debounce(({ status }) => {
      if (data) {
        updateData({ ...data, status })
      }
    }, 300)
  )

  // Update the resources (links) when a WS event is emitted
  useEmitter(
    getResourceRegistrationNotificationKey(deviceId),
    ({ event, resource }) => {
      if (data) {
        let updatedLinks = []

        if (event === resourceEventTypes.ADDED) {
          updatedLinks = data.links.concat(resource)
        } else {
          updatedLinks = data.links.filter(
            link => link.href !== resource.href
          )
        }

        updateData({
          ...data,
          links: updatedLinks,
        })
      }
    }
  )

  return { data, updateData, ...rest }
}
