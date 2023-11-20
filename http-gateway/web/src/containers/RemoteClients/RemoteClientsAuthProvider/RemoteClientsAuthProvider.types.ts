import { ReactElement } from 'react'

import { WellKnownConfigType } from '@shared-ui/common/hooks'
import { RemoteClientType } from '@shared-ui/app/clientApp/RemoteClients/RemoteClients.types'

export type Props = {
    children: (reInitializationLoading: boolean, initializationLoading: boolean, reInitializationError: boolean) => ReactElement
    clientData: RemoteClientType
    loading?: boolean
    reInitialization?: boolean
    setAuthError: (error: string) => void
    setInitialize: (isInitialize: boolean) => void
    unauthorizedCallback: () => void
    wellKnownConfig?: WellKnownConfigType
}
