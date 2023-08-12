export type Props = {
    buildInformation: {
        buildDate: string
        commitHash: string
        commitDate: string
        releaseUrl: string
        version: string
    }
    collapsed: boolean
    userData?: {
        profile?: {
            family_name?: string
            picture?: string
            name?: string
        }
    }
    setCollapsed: () => {}
    signOutRedirect?: (data: any) => {}
}
