import { ReactNode } from 'react'
import { resourceModalTypes } from '../../constants'

export type DevicesResourcesModalType = typeof resourceModalTypes[keyof typeof resourceModalTypes]

export type DevicesResourcesModalParamsType = {
    href: string
    currentInterface: string
}

export type Props = {
    confirmDisabled: boolean
    createResource: ({ href, currentInterface }: DevicesResourcesModalParamsType, jsonData?: any) => void
    data?: {
        deviceId?: string
        href?: string
        interfaces?: string[]
        types: string[]
    }
    deviceId?: string
    fetchResource: ({ href, currentInterface }: DevicesResourcesModalParamsType) => void | Promise<void>
    isDeviceOnline: boolean
    isUnregistered: boolean
    loading: boolean
    onClose: () => void
    resourceData?: {
        types: string[]
        data: {
            content: any
            status: string
        }
    }
    retrieving: boolean
    ttlControl?: ReactNode
    type?: DevicesResourcesModalType
    updateResource: ({ href, currentInterface }: DevicesResourcesModalParamsType, jsonData?: any) => void
}

export const defaultProps = {
    type: resourceModalTypes.UPDATE_RESOURCE,
}
