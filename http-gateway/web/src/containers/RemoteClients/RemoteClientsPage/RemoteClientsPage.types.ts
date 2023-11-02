import { ReactElement } from 'react'

export type Props = {
    children: (clientData: any, reInitializationError: boolean, reInitializationLoading: boolean, initializedByAnother: boolean) => ReactElement
}
