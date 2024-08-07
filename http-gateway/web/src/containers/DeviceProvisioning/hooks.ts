import { useCallback, useContext, useEffect, useState } from 'react'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'

import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { dpsApiEndpoints } from './constants'
import { pemToString } from '@/containers/DeviceProvisioning/utils'

const getConfig = () => security.getGeneralConfig() as SecurityConfig
const getWellKnow = () => security.getWellKnownConfig()

type EnrollmentGroupType = {
    attestationMechanism: any
    hubId: string
    id: string
    name: string
    owner: string
}

export const useProvisioningRecordsList = (): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    const [data, setData] = useState<any>(null)

    const {
        data: provisionRecordData,
        refresh: provisionRecordRefresh,
        loading: provisionRecordLoading,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.PROVISIONING_RECORDS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-provisioning-records',
        unauthorizedCallback,
    })

    const {
        data: enrollmentGroupsData,
        refresh: enrollmentGroupsRefresh,
        loading: enrollmentGroupsLoading,
    }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.ENROLLMENT_GROUPS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-enrollment-groups',
        unauthorizedCallback,
    })

    useEffect(() => {
        if (provisionRecordData && Array.isArray(provisionRecordData) && enrollmentGroupsData && !provisionRecordLoading && !enrollmentGroupsLoading) {
            setData(
                provisionRecordData?.map((provisioningRecord: any) => ({
                    ...provisioningRecord,
                    enrollmentGroupData: enrollmentGroupsData.find(
                        (enrollmentGroup: EnrollmentGroupType) => enrollmentGroup.id === provisioningRecord.enrollmentGroupId
                    ),
                }))
            )
        }
    }, [enrollmentGroupsData, enrollmentGroupsLoading, provisionRecordData, provisionRecordLoading])

    const refresh = useCallback(() => {
        provisionRecordRefresh()
        enrollmentGroupsRefresh()
    }, [provisionRecordRefresh, enrollmentGroupsRefresh])

    return { data, refresh, loading: provisionRecordLoading || enrollmentGroupsLoading, ...rest }
}

export const useProvisioningRecordsDetail = (provisioningRecordId?: string): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    const [data, setData] = useState<any>(null)

    const {
        data: provisionRecordData,
        refresh: provisioningRecordRefresh,
        loading: provisioningRecordLoading,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.PROVISIONING_RECORDS}?idFilter=${provisioningRecordId}`, {
        telemetryWebTracer,
        telemetrySpan: `get-provisioning-record-${provisioningRecordId}`,
        requestActive: !!provisioningRecordId,
        unauthorizedCallback,
    })

    const enrollmentGroupId = provisionRecordData ? provisionRecordData[0]?.enrollmentGroupId : ''

    const {
        data: enrollmentGroupsData,
        refresh: refreshEnrollmentGroup,
        loading: enrollmentGroupsLoading,
    }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.ENROLLMENT_GROUPS}?idFilter=${enrollmentGroupId}`, {
        telemetryWebTracer,
        requestActive: !!provisionRecordData && !provisioningRecordLoading,
        telemetrySpan: `get-enrollment-group-${enrollmentGroupId}`,
        unauthorizedCallback,
    })

    const idFilter = enrollmentGroupsData && enrollmentGroupsData[0] ? enrollmentGroupsData[0].hubIds.map((id: string) => `idFilter=${id}`).join('&') : ''
    const {
        data: hubsData,
        refresh: refreshHubs,
        loading: hubsLoading,
    }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.HUBS}?${idFilter}`, {
        telemetryWebTracer,
        requestActive: !!enrollmentGroupsData && !enrollmentGroupsLoading && !provisioningRecordLoading,
        telemetrySpan: `get-hubs-${idFilter}`,
        unauthorizedCallback,
    })

    useEffect(() => {
        if (provisionRecordData && Array.isArray(provisionRecordData) && !provisioningRecordLoading && !enrollmentGroupsLoading && !hubsLoading) {
            setData({ ...provisionRecordData[0], enrollmentGroupData: enrollmentGroupsData ? { ...enrollmentGroupsData[0], hubsData: hubsData || [] } : {} })
        }
    }, [enrollmentGroupsData, hubsData, provisionRecordData, provisioningRecordLoading, enrollmentGroupsLoading, hubsLoading])

    const refresh = useCallback(() => {
        provisioningRecordRefresh()
        refreshEnrollmentGroup()
        refreshHubs()
    }, [provisioningRecordRefresh, refreshEnrollmentGroup, refreshHubs])

    return { data, refresh, loading: provisioningRecordLoading || enrollmentGroupsLoading || hubsLoading, ...rest }
}

export const useEnrollmentGroupDataList = (): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    const [data, setData] = useState<any>(null)

    const {
        data: enrollmentGroupsData,
        refresh: refreshEnrollmentGroup,
        loading: enrollmentGroupsLoading,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.ENROLLMENT_GROUPS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-enrollment-groups-data',
        unauthorizedCallback,
    })

    const {
        data: hubsData,
        refresh: refreshHubs,
        loading: hubsLoading,
    }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.HUBS}`, {
        telemetryWebTracer,
        telemetrySpan: `get-hubs`,
        unauthorizedCallback,
    })

    useEffect(() => {
        if (enrollmentGroupsData && Array.isArray(enrollmentGroupsData) && !enrollmentGroupsLoading && !hubsLoading) {
            setData(
                enrollmentGroupsData.map((group: any) => ({
                    ...group,
                    hubsData: hubsData ? hubsData.filter((hubData: any) => group.hubIds.includes(hubData.id)) : [],
                }))
            )
        }
    }, [enrollmentGroupsData, enrollmentGroupsLoading, hubsData, hubsLoading])

    const refresh = useCallback(() => {
        refreshEnrollmentGroup()
        refreshHubs()
    }, [refreshEnrollmentGroup, refreshHubs])

    return { data, refresh, loading: enrollmentGroupsLoading || hubsLoading, ...rest }
}

export const useEnrollmentGroupDetail = (enrollmentGroupId?: string): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    const [data, setData] = useState(null)

    const {
        data: enrollmentGroupData,
        refresh: refreshEnrollmentGroup,
        loading: enrollmentGroupLoading,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.ENROLLMENT_GROUPS}?idFilter=${enrollmentGroupId}`, {
        telemetryWebTracer,
        telemetrySpan: `get-enrollment-group-${enrollmentGroupId}`,
        unauthorizedCallback,
    })

    const idFilter = enrollmentGroupData && enrollmentGroupData[0] ? enrollmentGroupData[0].hubIds.map((id: string) => `idFilter=${id}`).join('&') : ''
    const {
        data: hubsData,
        refresh: refreshHubs,
        loading: hubsLoading,
    }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.HUBS}?${idFilter}`, {
        telemetryWebTracer,
        requestActive: !!enrollmentGroupData && !enrollmentGroupLoading,
        telemetrySpan: `get-hubs-${idFilter}`,
        unauthorizedCallback,
    })

    const formatPSK = (psk?: string) => {
        if (!psk) {
            return ''
        }

        if (!psk.startsWith('/')) {
            return pemToString(psk)
        }

        return psk
    }

    useEffect(() => {
        if (!enrollmentGroupLoading && !hubsLoading && enrollmentGroupData && Array.isArray(enrollmentGroupData) && hubsData) {
            setData({
                ...enrollmentGroupData[0],
                preSharedKey: formatPSK(enrollmentGroupData[0]?.preSharedKey),
                hubsData,
            })
        }
    }, [enrollmentGroupData, enrollmentGroupLoading, hubsData, hubsLoading])

    const refresh = useCallback(() => {
        refreshEnrollmentGroup()
        refreshHubs()
    }, [refreshEnrollmentGroup, refreshHubs])

    return { data, refresh, loading: enrollmentGroupLoading || hubsLoading, ...rest }
}

export const useLinkedHubsList = (): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    return useStreamApi(`${url}${dpsApiEndpoints.HUBS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-hubs',
        unauthorizedCallback,
    })
}

export const useHubDetail = (hubId: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer, unauthorizedCallback } = useContext(AppContext)
    const url = getWellKnow()?.ui?.deviceProvisioningService || getConfig().httpGatewayAddress

    const [data, setData] = useState(null)

    const { data: hubsData, ...rest }: StreamApiPropsType = useStreamApi(`${url}${dpsApiEndpoints.HUBS}?idFilter=${hubId}`, {
        telemetryWebTracer,
        telemetrySpan: `get-hub-${hubId}`,
        requestActive,
        unauthorizedCallback,
    })

    useEffect(() => {
        if (hubsData && Array.isArray(hubsData)) {
            setData({
                ...hubsData[0],
            })
        }
    }, [hubsData])

    return { data, ...rest }
}
