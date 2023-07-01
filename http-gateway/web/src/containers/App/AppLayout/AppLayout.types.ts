export type Props = {
    collapsed: boolean
    userData?: {
        profile?: {
            family_name?: string
            picture?: string
            name?: string
        }
    }
    setCollapsed: () => boolean
}
