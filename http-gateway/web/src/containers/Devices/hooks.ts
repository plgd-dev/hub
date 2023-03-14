// @ts-nocheck
import debounce from 'lodash/debounce'
import { useStreamApi, useEmitter } from '@shared-ui/common/hooks'

import { devicesApiEndpoints, DEVICES_STATUS_WS_KEY, resourceEventTypes } from './constants'
import { updateDevicesDataStatus, getResourceRegistrationNotificationKey } from './utils'
import { security } from '@shared-ui/common/services'
import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'

const getConfig = () => security.getGeneralConfig() as SecurityConfig

export const useDevicesList = () => {
    const { data, updateData, ...rest } = useStreamApi(`${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES}`, { telemetrySpan: 'get-devices' })

    // Update the metadata when a WS event is emitted
    useEmitter(DEVICES_STATUS_WS_KEY, (newDeviceStatus: any) => {
        if (data) {
            // Update the data with the current device status and twinSynchronization
            updateData(updateDevicesDataStatus(data, newDeviceStatus))
        }
    })

    return { data, updateData, ...rest }
}

export const useDeviceDetails = (deviceId: string) => {
    const { data, updateData, ...rest }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}`, {
        streamApi: false,
        telemetrySpan: 'get-device-detail',
    })

    // Update the metadata when a WS event is emitted
    useEmitter(
        `${DEVICES_STATUS_WS_KEY}.${deviceId}`,
        debounce(({ status, twinEnabled }) => {
            if (data) {
                updateData({
                    ...data,
                    metadata: {
                        ...data.metadata,
                        twinEnabled,
                        connection: {
                            ...data.metadata.connection,
                            status: status,
                        },
                    },
                })
            }
        }, 300)
    )

    return { data, updateData, ...rest }
}

export const useDevicesResources = (deviceId: string) => {
    const { data, updateData, ...rest }: StreamApiPropsType = useStreamApi(
        `${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES_RESOURCES}?device_id_filter=${deviceId}`,
        { telemetrySpan: 'get-device-resources' }
    )

    useEmitter(getResourceRegistrationNotificationKey(deviceId), ({ event, resources: updatedResources }) => {
        if (data?.[0]?.resources) {
            const resources = data[0].resources // get the first set of resources from an array, since it came from a stream of data
            let updatedLinks = []

            updatedResources.forEach((resource) => {
                if (event === resourceEventTypes.ADDED) {
                    const linkExists = resources.findIndex((link) => link.href === resource.href) !== -1
                    if (linkExists) {
                        // Already exists, update
                        updatedLinks = resources.map((link) => {
                            if (link.href === resource.href) {
                                return resource
                            }

                            return link
                        })
                    } else {
                        updatedLinks = resources.concat(resource)
                    }
                } else {
                    updatedLinks = resources.filter((link) => link.href !== resource.href)
                }
            })

            updateData([{ ...data[0], resources: updatedLinks }])
        }
    })

    return { data, updateData, ...rest }
}
