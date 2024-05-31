export type Props = {
    deviceName: string
    deviceStatus: string
    isActiveTab: boolean
    isOnline: boolean
    isUnregistered: boolean
    loading: boolean
    loadingResources?: boolean
    refreshResources: () => void
    resourcesData?: any
}
