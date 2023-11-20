import { BuildInformationType } from '@shared-ui/common/hooks'

export type AppContextType = {
    buildInformation?: BuildInformationType | null
    collapsed?: boolean
    footerExpanded?: boolean
    setFooterExpanded?: (expand: boolean) => void
    setTheme?: (theme: string) => void
    telemetryWebTracer?: any
    unauthorizedCallback?: () => void
}
