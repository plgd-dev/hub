import debounce from 'lodash/debounce'

import { useApi, useStreamApi } from '@/common/hooks'
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
  const { data, updateData, ...rest } = useStreamApi(
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
        updateData({
          ...data,
          metadata: {
            ...data.metadata,
            status: {
              ...data.metadata.status,
              value: status,
            },
          },
        })
      }
    }, 300)
  )

  return { data, updateData, ...rest }
}

export const useThingsResources = deviceId => {
  const { httpGatewayAddress } = useAppConfig()

  // Fetch the data
  const { data, updateData, ...rest } = useStreamApi(
    `${httpGatewayAddress}${
      thingsApiEndpoints.THINGS_RESOURCES
    }?device_id_filter=${deviceId}`
  )

  useEmitter(
    getResourceRegistrationNotificationKey(deviceId),
    ({ event, resource }) => {
      if (data?.[0]?.resources) {
        const resources = data[0].resources // get the first set of resources from an array, since it came from a stream of data
        let updatedLinks = []

        if (event === resourceEventTypes.ADDED) {
          const linkExists =
            resources.findIndex(link => link.href === resource.href) !== -1
          if (linkExists) {
            // Already exists, update
            updatedLinks = resources.map(link => {
              if (link.href === resource.href) {
                return resource
              }

              return link
            })
          } else {
            updatedLinks = resources.concat(resource)
          }
        } else {
          updatedLinks = resources.filter(link => link.href !== resource.href)
        }

        updateData([{ ...data[0], resources: updatedLinks }])
      }
    }
  )

  return { data, updateData, ...rest }
}
