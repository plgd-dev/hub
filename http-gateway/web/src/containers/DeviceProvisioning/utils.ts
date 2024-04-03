import { parseCertificate } from '@shared-ui/common/services/certificates'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import { provisioningStatuses } from '@/containers/DeviceProvisioning/constants'
import { CA_BASE64_PREFIX } from '@shared-ui/components/Organisms/CaPool'
import { DataType } from '@/containers/DeviceProvisioning/ProvisioningRecords/ProvisioningRecordsListPage.types'

export const getStatusFromCode = (code: number) => ([67, 68, 69, 95].includes(code) ? provisioningStatuses.SUCCESS : provisioningStatuses.ERROR)

export const getProvisioningRecordStatus = (data: DataType) => {
    const statuses = [
        getStatusFromCode(data.acl.status.coapCode),
        getStatusFromCode(data.cloud.status.coapCode),
        getStatusFromCode(data.ownership.status.coapCode),
        getStatusFromCode(data.plgdTime.coapCode),
        getStatusFromCode(data.credential.status.coapCode),
    ]

    if (statuses.some((status) => status === provisioningStatuses.ERROR)) {
        return provisioningStatuses.ERROR
    }

    return provisioningStatuses.SUCCESS
}

export const getStatusFromData = (data: any) => {
    const provisioningRecordStatus = getProvisioningRecordStatus(data)

    if (provisioningRecordStatus === provisioningStatuses.ERROR) {
        return provisioningStatuses.ERROR
    }

    // some certificate can be expired
    if (data.parsedData.some((p: { status: boolean }) => !p.status)) {
        return provisioningStatuses.WARNING
    }

    return provisioningStatuses.SUCCESS
}

export type CertDataType = {
    usage: string
    publicData?: {
        data: string
        encoding: string
    }
}

type OptionsType = {
    errorTitle: string
    errorId: string
}

export const parseCerts = async (certs: any, options: OptionsType) => {
    const parsed = certs?.map(async (certsData: CertDataType, key: number) => {
        try {
            const { usage, publicData } = certsData

            if (usage !== 'NONE' && publicData) {
                return await parseCertificate(atob(publicData.data), key, certsData)
            } else {
                return null
            }
        } catch (e: any) {
            let error = e
            if (!(error instanceof Error)) {
                error = new Error(e)
            }

            console.error(error)
            Notification.error({ title: options.errorTitle, message: error.message }, { notificationId: options.errorId })
        }
    })

    return await Promise.all(parsed)
}

export function nameLengthValidator(file: any, privateKey = false) {
    const format = file.name.split('.').pop()

    if ((privateKey && !['pem', 'key'].includes(format)) || (!privateKey && !['pem', 'crt', 'cer'].includes(format))) {
        return {
            code: 'invalid-format',
            message: `Bad file format`,
        }
    }
    return null
}

export const stringToPem = (str: string) => `${CA_BASE64_PREFIX}${btoa(str)}`

export const pemToString = (pem: string) => atob(pem.replace(CA_BASE64_PREFIX, ''))
