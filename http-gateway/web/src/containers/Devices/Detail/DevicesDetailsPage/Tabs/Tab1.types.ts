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
    softwareUpdateData?: {
        // https://github.com/openconnectivityfoundation/core-extensions/blob/master/swagger2.0/oic.r.softwareupdate.swagger.json
        swupdateaction: string // https://github.com/plgd-dev/device/blob/2a60018de0639e7f225254ff9487bcf91bbb603f/schema/softwareupdate/swupdate.go#L49
        swupdateresult: number
        swupdatestate: string // https://github.com/plgd-dev/device/blob/2a60018de0639e7f225254ff9487bcf91bbb603f/schema/softwareupdate/swupdate.go#L58
        updatetime: string // "2023-06-02T07:37:00Z", // when the action will be performed
        lastupdate: string // "2023-06-02T07:37:02.330206Z",  // when the upgrade was performed
        nv: string // "0.0.12", // new available version
        purl: string // url with query parameters to update server
        signed: string // "vendor"
    }
    endpoints?: string[]
}
