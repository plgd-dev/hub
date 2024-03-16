import { useCallback, useContext, useEffect, useState } from 'react'

import AppContext from '@shared-ui/app/share/AppContext'
import { useStreamApi } from '@shared-ui/common/hooks'
import { security } from '@shared-ui/common/services'

import { SecurityConfig, StreamApiPropsType } from '@/containers/App/App.types'
import { dpsApiEndpoints } from './constants'

const getConfig = () => security.getGeneralConfig() as SecurityConfig

type EnrollmentGroupType = {
    attestationMechanism: any
    hubId: string
    id: string
    name: string
    owner: string
}

export const useProvisioningRecordsList = (): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    const [data, setData] = useState<any>(null)

    const {
        data: provisionRecordData,
        refresh: provisionRecordRefresh,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.PROVISIONING_RECORDS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-provisioning-records',
    })

    const { data: enrollmentGroupsData, refresh: enrollmentGroupsRefresh }: StreamApiPropsType = useStreamApi(
        `${getConfig().httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}`,
        {
            telemetryWebTracer,
            telemetrySpan: 'get-enrollment-groups',
        }
    )

    useEffect(() => {
        if (provisionRecordData && Array.isArray(provisionRecordData)) {
            setData(
                provisionRecordData?.map((provisioningRecord: any) => ({
                    ...provisioningRecord,
                    enrollmentGroupData: enrollmentGroupsData.find(
                        (enrollmentGroup: EnrollmentGroupType) => enrollmentGroup.id === provisioningRecord.enrollmentGroupId
                    ),
                }))
            )
        }
    }, [enrollmentGroupsData, provisionRecordData])

    const refresh = useCallback(() => {
        provisionRecordRefresh()
        enrollmentGroupsRefresh()
    }, [provisionRecordRefresh, enrollmentGroupsRefresh])

    return { data, refresh, ...rest }
}

export const useProvisioningRecordsDetail = (provisioningRecordId?: string): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    const [data, setData] = useState<any>(null)

    const {
        data: provisionRecordData,
        refresh: provisioningRecordRefresh,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.PROVISIONING_RECORDS}?idFilter=${provisioningRecordId}`, {
        telemetryWebTracer,
        telemetrySpan: `get-provisioning-record-${provisioningRecordId}`,
        requestActive: !!provisioningRecordId,
    })

    const enrollmentGroupId = data ? data[0].enrollmentGroupId : ''

    const { data: enrollmentGroupsData, refresh: refreshEnrollmentGroup }: StreamApiPropsType = useStreamApi(
        `${getConfig().httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}?idFilter=${enrollmentGroupId}`,
        {
            telemetryWebTracer,
            telemetrySpan: `get-enrollment-group-${enrollmentGroupId}`,
        }
    )

    const idFilter = enrollmentGroupsData && enrollmentGroupsData[0] ? enrollmentGroupsData[0].hubIds.map((id: string) => `idFilter=${id}`).join('&') : ''
    const { data: hubsData, refresh: refreshHubs }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.HUBS}?${idFilter}`, {
        telemetryWebTracer,
        telemetrySpan: `get-hubs-${idFilter}`,
    })

    useEffect(() => {
        if (provisionRecordData && Array.isArray(provisionRecordData)) {
            setData({ ...provisionRecordData[0], enrollmentGroupData: enrollmentGroupsData ? { ...enrollmentGroupsData[0], hubsData: hubsData || [] } : {} })
        }
    }, [enrollmentGroupsData, hubsData, provisionRecordData, provisioningRecordRefresh])

    const refresh = useCallback(() => {
        provisioningRecordRefresh()
        refreshEnrollmentGroup()
        refreshHubs()
    }, [provisioningRecordRefresh, refreshEnrollmentGroup, refreshHubs])

    return { data, refresh, ...rest }
}

export const useEnrollmentGroupDataList = (): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    const [data, setData] = useState<any>(null)

    const {
        data: enrollmentGroupsData,
        refresh: refreshEnrollmentGroup,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-enrollment-groups-data',
    })

    const { data: hubsData, refresh: refreshHubs }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.HUBS}`, {
        telemetryWebTracer,
        telemetrySpan: `get-hubs`,
    })

    useEffect(() => {
        if (enrollmentGroupsData && Array.isArray(enrollmentGroupsData)) {
            setData(
                enrollmentGroupsData.map((group: any) => ({
                    ...group,
                    hubsData: hubsData ? hubsData.filter((hubData: any) => group.hubIds.includes(hubData.id)) : [],
                }))
            )
        }
    }, [enrollmentGroupsData, hubsData])

    const refresh = useCallback(() => {
        refreshEnrollmentGroup()
        refreshHubs()
    }, [refreshEnrollmentGroup, refreshHubs])

    return { data, refresh, ...rest }
}

export const useEnrollmentGroupDetail = (enrollmentGroupId?: string): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    const [data, setData] = useState(null)

    const {
        data: enrollmentGroupData,
        refresh: refreshEnrollmentGroup,
        loading: enrollmentGroupLoading,
        ...rest
    }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}?idFilter=${enrollmentGroupId}`, {
        telemetryWebTracer,
        telemetrySpan: `get-enrollment-group-${enrollmentGroupId}`,
    })

    const idFilter = enrollmentGroupData && enrollmentGroupData[0] ? enrollmentGroupData[0].hubIds.map((id: string) => `idFilter=${id}`).join('&') : ''
    const {
        data: hubsData,
        refresh: refreshHubs,
        loading: hubsLoading,
    }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.HUBS}?${idFilter}`, {
        telemetryWebTracer,
        requestActive: !!enrollmentGroupData && !enrollmentGroupLoading,
        telemetrySpan: `get-hubs-${idFilter}`,
    })

    useEffect(() => {
        if (enrollmentGroupData && Array.isArray(enrollmentGroupData) && !enrollmentGroupLoading && !hubsLoading) {
            setData({ ...enrollmentGroupData[0], hubsData })
        }
    }, [enrollmentGroupData, enrollmentGroupLoading, hubsData, hubsLoading])

    const refresh = useCallback(() => {
        refreshEnrollmentGroup()
        refreshHubs()
    }, [refreshEnrollmentGroup, refreshHubs])

    return { data, refresh, loading: enrollmentGroupLoading || hubsLoading, ...rest }
}

export const useLinkedHubsList = (): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.HUBS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-hubs',
    })
}

export const useHubDetail = (hubId: string, requestActive = false): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    const [data, setData] = useState(null)

    const { data: hubsData, ...rest }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.HUBS}?idFilter=${hubId}`, {
        telemetryWebTracer,
        telemetrySpan: `get-hub-${hubId}`,
        requestActive,
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
