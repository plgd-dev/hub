import { BuildInformationType } from '@shared-ui/common/hooks'

export type AppContextType = {
    buildInformation?: BuildInformationType | null
    collapsed?: boolean
    unauthorizedCallback?: () => void
    footerExpanded: boolean
    telemetryWebTracer?: any
    setFooterExpanded?: (expand: boolean) => void
}
