import { WellKnownConfigType } from '@shared-ui/common/hooks'
import { PlgdThemeType } from '@shared-ui/components/Atomic/_theme'

export type Props = {
    collapsed: boolean
    openTelemetry: any
    setCollapsed: () => {}
    wellKnownConfig: WellKnownConfigType
}
