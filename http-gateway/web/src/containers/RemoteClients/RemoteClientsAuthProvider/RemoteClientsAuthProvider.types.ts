import { WellKnownConfigType } from '@shared-ui/common/hooks'
import { ReactElement } from 'react'

export type AppAuthProviderRefType = {
    getSignOutMethod(): any
    getUserData(): any
}

export type Props = {
    children: ReactElement
    setAuthError: (error: string) => void
    setInitialize: (isInitialize?: boolean) => void
    wellKnownConfig?: WellKnownConfigType
}
