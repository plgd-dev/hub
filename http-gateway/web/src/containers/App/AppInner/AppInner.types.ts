import { WellKnownConfigType } from '@shared-ui/common/hooks'

export type Props = {
    collapsed: boolean
    openTelemetry: any
    setCollapsed: (c: boolean) => void
    setStoredPathname: (pathname: string) => void
    wellKnownConfig: WellKnownConfigType
}
