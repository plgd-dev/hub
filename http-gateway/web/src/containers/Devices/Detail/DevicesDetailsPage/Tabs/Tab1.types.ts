export type Props = {
    deviceId: string
    deviceName: string
    firmware?: string
    model?: string
    pendingCommandsData?: []
    isActiveTab: boolean
    isTwinEnabled: boolean
    setTwinSynchronization: (newTwinEnabled: boolean) => Promise<void>
    twinSyncLoading: boolean
    types?: string[]
}
