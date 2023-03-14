export type Props = {
    types?: string[]
    deviceId: string
    isTwinEnabled: boolean
    setTwinSynchronization: (newTwinEnabled: boolean) => Promise<void>
    twinSyncLoading: boolean
}
