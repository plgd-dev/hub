export type Props = {
    deviceName: string
    deviceStatus: string
    isActiveTab: boolean
    isOnline: boolean
    isUnregistered: boolean
    loading: boolean
    loadingResources?: boolean
    pageSize: {
        height?: number
        width?: number
    }
    refreshResources: () => void
    resourcesData?: any
}
