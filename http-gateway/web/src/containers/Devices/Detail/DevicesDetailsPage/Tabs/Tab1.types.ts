export type Props = {
    types?: string[]
    deviceId: string
    deviceName: string
    isActiveTab: boolean
    isTwinEnabled: boolean
    setTwinSynchronization: (newTwinEnabled: boolean) => Promise<void>
    twinSyncLoading: boolean
}
