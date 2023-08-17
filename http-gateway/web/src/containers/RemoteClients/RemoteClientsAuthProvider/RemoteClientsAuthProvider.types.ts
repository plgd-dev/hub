import { ReactElement } from 'react'

import { WellKnownConfigType } from '@shared-ui/common/hooks'
import { RemoteClientType } from '@shared-ui/app/clientApp/RemoteClients/RemoteClients.types'

export type AppAuthProviderRefType = {
    getSignOutMethod(): any
    getUserData(): any
}

export type Props = {
    children: ReactElement
    clientData: RemoteClientType
    setAuthError: (error: string) => void
    setInitialize: (isInitialize?: boolean) => void
    wellKnownConfig?: WellKnownConfigType
}
