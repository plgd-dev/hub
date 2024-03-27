export type Props = {
    buildInformation: {
        buildDate: string
        commitHash: string
        commitDate: string
        releaseUrl: string
        version: string
    }
    collapsed: boolean
    mockApp?: boolean
    setCollapsed: (c: boolean) => void
    setStoredPathname?: (pathname: string) => void
    signOutRedirect?: (data: any) => {}
    userData?: {
        profile?: {
            family_name?: string
            picture?: string
            name?: string
        }
    }
}
