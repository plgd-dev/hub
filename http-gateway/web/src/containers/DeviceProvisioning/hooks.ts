import { useContext } from 'react'

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

    const { data, ...rest }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.PROVISIONING_RECORDS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-provisioning-records',
    })

    const { data: enrollmentGroupsData }: StreamApiPropsType = useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-enrollment-groups',
    })

    if (data && enrollmentGroupsData) {
        const dataForUpdate = data?.map((provisioningRecord: any) => ({
            ...provisioningRecord,
            enrollmentGroupData: enrollmentGroupsData.find(
                (enrollmentGroup: EnrollmentGroupType) => enrollmentGroup.id === provisioningRecord.enrollmentGroupId
            ),
        }))

        if (dataForUpdate) {
            return { data: dataForUpdate, ...rest }
        }
    }

    return { data, ...rest }
}

export const useProvisioningRecordsDetail = (provisioningRecordId?: string): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    const { data, refresh, ...rest }: StreamApiPropsType = useStreamApi(
        `${getConfig().httpGatewayAddress}${dpsApiEndpoints.PROVISIONING_RECORDS}?idFilter=${provisioningRecordId}`,
        {
            telemetryWebTracer,
            telemetrySpan: `get-provisioning-record-${provisioningRecordId}`,
            requestActive: !!provisioningRecordId,
        }
    )

    const enrollmentGroupId = data ? data[0].enrollmentGroupId : ''

    const { data: enrollmentGroupsData, refresh: refreshEnrollmentGroup }: StreamApiPropsType = useStreamApi(
        `${getConfig().httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}?idFilter=${enrollmentGroupId}`,
        {
            telemetryWebTracer,
            telemetrySpan: `get-enrollment-group-${enrollmentGroupId}`,
        }
    )

    if (data && enrollmentGroupsData) {
        return {
            data: { ...data[0], enrollmentGroupData: enrollmentGroupsData[0] },
            refresh: () => {
                refresh()
                refreshEnrollmentGroup()
            },
            ...rest,
        }
    }

    return { data, refresh, ...rest }
}

export const useEnrollmentGroupDataList = (): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-enrollment-groups-data',
    })
}

export const useEnrollmentGroupDetail = (enrollmentGroupId?: string): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    const { data, ...rest }: StreamApiPropsType = useStreamApi(
        `${getConfig().httpGatewayAddress}${dpsApiEndpoints.ENROLLMENT_GROUPS}?idFilter=${enrollmentGroupId}`,
        {
            telemetryWebTracer,
            telemetrySpan: `get-enrollment-group-${enrollmentGroupId}`,
        }
    )

    if (data) {
        return {
            data: data[0],
            ...rest,
        }
    }

    return { data, ...rest }
}

export const useLinkedHubsList = (): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.HUBS}`, {
        telemetryWebTracer,
        telemetrySpan: 'get-hubs',
    })
}

export const useHubDetail = (hubId: string): StreamApiPropsType => {
    const { telemetryWebTracer } = useContext(AppContext)

    return useStreamApi(`${getConfig().httpGatewayAddress}${dpsApiEndpoints.HUBS}?idFilter=${hubId}}`, {
        telemetryWebTracer,
        telemetrySpan: `get-hub-${hubId}`,
    })
}
