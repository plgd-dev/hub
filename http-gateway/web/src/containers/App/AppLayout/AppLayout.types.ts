import { PlgdThemeType } from '@shared-ui/components/Atomic/_theme'

export type Props = {
    buildInformation: {
        buildDate: string
        commitHash: string
        commitDate: string
        releaseUrl: string
        version: string
    }
    collapsed: boolean
    setCollapsed: () => {}
    signOutRedirect?: (data: any) => {}
    theme: PlgdThemeType
    userData?: {
        profile?: {
            family_name?: string
            picture?: string
            name?: string
        }
    }
}
