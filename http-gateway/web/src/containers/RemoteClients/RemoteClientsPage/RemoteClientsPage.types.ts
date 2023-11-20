import { ReactElement } from 'react'

export type Props = {
    children: (clientData: any, reInitializationError: boolean, loading: boolean, initializedByAnother: boolean) => ReactElement
}
