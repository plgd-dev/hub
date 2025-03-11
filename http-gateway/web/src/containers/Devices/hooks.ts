// @ts-nocheck
import { useContext, useEffect, useState } from 'react'
import debounce from 'lodash/debounce'

import { useStreamApi, useEmitter } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'
import AppContext from '@shared-ui/app/share/AppContext'

import { devicesApiEndpoints, DEVICES_STATUS_WS_KEY, resourceEventTypes } from './constants'
import { updateDevicesDataStatus, getResourceRegistrationNotificationKey } from './utils'
import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'

const getConfig = () => security.getGeneralConfig() as SecurityConfig
const getWellKnow = () => security.getWellKnownConfig()

export const useDevicesList = (requestActive = true): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)

    const { data, updateData, setState, ...rest }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-devices',
        unauthorizedCallback,
        requestActive,
    })

    // Update the metadata when a WS event is emitted
    useEmitter(DEVICES_STATUS_WS_KEY, (newDeviceData: any) => {
        if (data) {
            // Update the data with the current device status and twinSynchronization
            // update data based on prevState
            setState((prevState) => ({ ...prevState, data: updateDevicesDataStatus(prevState.data, newDeviceData) }))
        }
    })

    return { data, updateData, ...rest }
}

export const useDeviceDetails = (deviceId: string) => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const { data, updateData, ...rest }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}`, {
        streamApi: false,
        telemetryWebTracer,
        telemetrySpan: 'get-device-detail',
        unauthorizedCallback,
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

export const useDeviceSoftwareUpdateDetails = (deviceId: string): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    return useStreamApi(`${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}/resources?type=oic.r.softwareupdate`, {
        streamApi: false,
        telemetryWebTracer,
        telemetrySpan: 'get-device-software-update-detail',
        unauthorizedCallback,
    })
}

export const useDevicesResources = (deviceId: string): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const { data, updateData, ...rest }: StreamApiPropsType = useStreamApi(
        `${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES_RESOURCES}?device_id_filter=${deviceId}`,
        { telemetryWebTracer, telemetrySpan: 'get-device-resources', unauthorizedCallback }
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

export const useDevicePendingCommands = (deviceId: string): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    return useStreamApi(`${getConfig().httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}/pending-commands`, {
        telemetryWebTracer,
        telemetrySpan: `get-device-pending-commands-${deviceId}`,
        unauthorizedCallback,
    })
}

export const useDeviceCertificates = (deviceId: string): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.certificateAuthority || getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress
    return useStreamApi(`${url}/certificate-authority/api/v1/signing/records?deviceIdFilter=${deviceId}`, {
        telemetryWebTracer,
        telemetrySpan: `get-device-certificates-${deviceId}`,
        unauthorizedCallback,
    })
}

export const useDeviceProvisioningRecord = (deviceId: string): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    const [data, setData] = useState(null)

    const { data: provisioningRecordData, ...rest }: StreamApiPropsType = useStreamApi(`${url}/api/v1/provisioning-records?deviceIdFilter=${deviceId}`, {
        telemetryWebTracer,
        telemetrySpan: `get-device-provisioning-record-${deviceId}`,
        unauthorizedCallback,
    })

    useEffect(() => {
        if (provisioningRecordData && Array.isArray(provisioningRecordData)) {
            setData({
                ...provisioningRecordData[0],
            })
        }
    }, [provisioningRecordData])

    return { data, ...rest }
}
